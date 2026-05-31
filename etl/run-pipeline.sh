#!/bin/bash
# ETL Pipeline Orchestrator
# Usage: ./run-pipeline.sh --disc-label DISC_001 --title "Movie Title" [--node bluray-node] [--skip-extract] [--skip-thumbnail]
set -euo pipefail

DISC_LABEL=""
CONTENT_TITLE=""
BLURAY_NODE=""
SKIP_EXTRACT=false
SKIP_THUMBNAIL=false
NAMESPACE="${NAMESPACE:-pechka}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
REGISTRY="${REGISTRY:-ghcr.io/k-wa-wa/pechka}"

usage() {
    echo "Usage: $0 --disc-label <label> --title <title> [--node <k8s-node>] [--skip-extract] [--skip-thumbnail]"
    exit 1
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --disc-label) DISC_LABEL="$2"; shift 2 ;;
        --title) CONTENT_TITLE="$2"; shift 2 ;;
        --node) BLURAY_NODE="$2"; shift 2 ;;
        --skip-extract) SKIP_EXTRACT=true; shift ;;
        --skip-thumbnail) SKIP_THUMBNAIL=true; shift ;;
        *) usage ;;
    esac
done

[ -z "$DISC_LABEL" ] && usage
[ -z "$CONTENT_TITLE" ] && usage

SAFE_LABEL=$(echo "$DISC_LABEL" | tr '[:upper:]' '[:lower:]' | tr '_' '-' | tr '.' '-')
TIMESTAMP=$(date +%s)

# Detect Kubernetes context to automatically set dev/prod defaults
CURRENT_CONTEXT=$(kubectl config current-context || echo "")
if [[ "$CURRENT_CONTEXT" == *kind* ]]; then
    echo "Detected local Kind context: $CURRENT_CONTEXT"
    DB_HOST="${DB_HOST:-postgres}"
    IMAGE_PULL_POLICY="IfNotPresent"
else
    echo "Using production context: $CURRENT_CONTEXT"
    DB_HOST="${DB_HOST:-primary.pg-cluster.svc.cluster.local}"
    IMAGE_PULL_POLICY="Always"
fi

echo "=== Pechka ETL Pipeline ==="
echo "Disc: $DISC_LABEL"
echo "Title: $CONTENT_TITLE"
echo "Namespace: $NAMESPACE"
echo "DB Host: $DB_HOST"
echo "Image Pull Policy: $IMAGE_PULL_POLICY"

# Phase 1: Extract (Bluray → NFS mkv/)
if [ "$SKIP_EXTRACT" = false ]; then
    echo ""
    echo "--- Phase 1: Extract ---"
    JOB_NAME="bluray-extract-${SAFE_LABEL}-${TIMESTAMP}"
    echo "Creating extract job: $JOB_NAME"

    python3 ./etl/instantiate-job.py \
        --placeholder bluray-extract \
        --name "$JOB_NAME" \
        --namespace "$NAMESPACE" \
        --image "${REGISTRY}/etl-extract:${IMAGE_TAG}" \
        --image-pull-policy "$IMAGE_PULL_POLICY" \
        --env DB_HOST="$DB_HOST" | kubectl apply -n "$NAMESPACE" -f -

    if [ -n "$BLURAY_NODE" ]; then
        kubectl patch job "$JOB_NAME" \
            --namespace "$NAMESPACE" \
            --type=merge \
            -p "{\"spec\":{\"template\":{\"spec\":{\"nodeName\":\"${BLURAY_NODE}\"}}}}"
    fi

    echo "Waiting for extract job to complete..."
    kubectl wait job/"$JOB_NAME" --for=condition=complete --timeout=3600s --namespace "$NAMESPACE"
    echo "Extract complete."
fi

# Phase 2: Transcode (MKV → HLS)
echo ""
echo "--- Phase 2: Transcode ---"
TRANSCODE_JOBS=()
MODES=(video audio)

for MODE in "${MODES[@]}"; do
    JOB_NAME="transcode-${SAFE_LABEL}-${MODE}-${TIMESTAMP}"
    echo "Creating transcode job: $JOB_NAME (mode=$MODE)"

    python3 ./etl/instantiate-job.py \
        --placeholder transcode-placeholder \
        --name "$JOB_NAME" \
        --namespace "$NAMESPACE" \
        --image "${REGISTRY}/etl-transform:${IMAGE_TAG}" \
        --image-pull-policy "$IMAGE_PULL_POLICY" \
        --set-arg="-input=/mnt/bluray/mkv/${DISC_LABEL}/title_00.mkv" \
        --set-arg="-output=/mnt/hls/${DISC_LABEL}" \
        --set-arg="-mode=${MODE}" | kubectl apply -n "$NAMESPACE" -f -

    TRANSCODE_JOBS+=("$JOB_NAME")
done

echo "Waiting for all transcode jobs to complete..."
for JOB in "${TRANSCODE_JOBS[@]}"; do
    kubectl wait job/"$JOB" --for=condition=complete --timeout=7200s --namespace "$NAMESPACE"
    echo "$JOB complete."
done

# Phase 3: Load (Register content & Upload HLS to MinIO)
echo ""
echo "--- Phase 3: Load ---"
LOAD_JOB="load-hls-${SAFE_LABEL}-${TIMESTAMP}"
echo "Creating load job: $LOAD_JOB"

python3 ./etl/instantiate-job.py \
    --placeholder load-hls-placeholder \
    --name "$LOAD_JOB" \
    --namespace "$NAMESPACE" \
    --image "${REGISTRY}/etl-load:${IMAGE_TAG}" \
    --image-pull-policy "$IMAGE_PULL_POLICY" \
    --env DB_HOST="$DB_HOST" \
    --env DISC_LABEL="$DISC_LABEL" \
    --env CONTENT_TITLE="$CONTENT_TITLE" | kubectl apply -n "$NAMESPACE" -f -

echo "Waiting for load job to complete..."
kubectl wait job/"$LOAD_JOB" --for=condition=complete --timeout=600s --namespace "$NAMESPACE"
echo "Load job complete."

# Parse IDs from load logs
LOAD_LOGS=$(kubectl logs job/"$LOAD_JOB" --namespace "$NAMESPACE")
CONTENT_ID=$(echo "$LOAD_LOGS" | grep -oE 'id=[a-zA-Z0-9-]+' | head -n 1 | cut -d= -f2)
SHORT_ID=$(echo "$LOAD_LOGS" | grep -oE 'short_id=[a-zA-Z0-9-]+' | head -n 1 | cut -d= -f2)

if [ -z "$CONTENT_ID" ] || [ -z "$SHORT_ID" ]; then
    echo "Error: Failed to parse content_id or short_id from load logs!"
    echo "Logs were:"
    echo "$LOAD_LOGS"
    exit 1
fi
echo "Parsed Content ID: $CONTENT_ID, Short ID: $SHORT_ID"

# Phase 4: Thumbnail
if [ "$SKIP_THUMBNAIL" = false ]; then
    echo ""
    echo "--- Phase 4: Thumbnail ---"
    THUMB_JOB="thumbnail-${SAFE_LABEL}-${TIMESTAMP}"
    echo "Creating thumbnail job: $THUMB_JOB"

    python3 ./etl/instantiate-job.py \
        --placeholder thumbnail-placeholder \
        --name "$THUMB_JOB" \
        --namespace "$NAMESPACE" \
        --image "${REGISTRY}/etl-thumbnail:${IMAGE_TAG}" \
        --image-pull-policy "$IMAGE_PULL_POLICY" \
        --env DB_HOST="$DB_HOST" \
        --set-arg="--input=/mnt/nfs/mkv/${DISC_LABEL}/title_00.mkv" \
        --set-arg="--short-id=${SHORT_ID}" \
        --set-arg="--content-id=${CONTENT_ID}" | kubectl apply -n "$NAMESPACE" -f -

    echo "Waiting for thumbnail job to complete..."
    kubectl wait job/"$THUMB_JOB" --for=condition=complete --timeout=1800s --namespace "$NAMESPACE"
    echo "Thumbnail job complete."
else
    echo "Skipping thumbnail step (--skip-thumbnail)"
fi

# Phase 5: Refresh Playlist
echo ""
echo "--- Phase 5: Refresh Playlist ---"
REFRESH_JOB="refresh-playlist-${SAFE_LABEL}-${TIMESTAMP}"
echo "Creating refresh-playlist job: $REFRESH_JOB"

python3 ./etl/instantiate-job.py \
    --placeholder refresh-latest-playlist \
    --name "$REFRESH_JOB" \
    --namespace "$NAMESPACE" \
    --image "${REGISTRY}/etl-refresh-latest-playlist:${IMAGE_TAG}" \
    --image-pull-policy "$IMAGE_PULL_POLICY" \
    --env DB_HOST="$DB_HOST" | kubectl apply -n "$NAMESPACE" -f -

echo "Waiting for refresh-playlist job to complete..."
kubectl wait job/"$REFRESH_JOB" --for=condition=complete --timeout=300s --namespace "$NAMESPACE"
echo "Refresh-playlist job complete."

echo ""
echo "=== Pipeline complete! ==="
echo "Content short_id: ${SHORT_ID}"
