#!/bin/bash
# ETL Pipeline Orchestrator
# Usage: ./run-pipeline.sh --disc-label DISC_001 --title "Movie Title" [--node bluray-node] [--skip-extract]
set -euo pipefail

DISC_LABEL=""
CONTENT_TITLE=""
BLURAY_NODE=""
SKIP_EXTRACT=false
NAMESPACE="${NAMESPACE:-pechka}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
REGISTRY="${REGISTRY:-ghcr.io/k-wa-wa/pechka}"

usage() {
    echo "Usage: $0 --disc-label <label> --title <title> [--node <k8s-node>] [--skip-extract]"
    exit 1
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --disc-label) DISC_LABEL="$2"; shift 2 ;;
        --title) CONTENT_TITLE="$2"; shift 2 ;;
        --node) BLURAY_NODE="$2"; shift 2 ;;
        --skip-extract) SKIP_EXTRACT=true; shift ;;
        *) usage ;;
    esac
done

[ -z "$DISC_LABEL" ] && usage
[ -z "$CONTENT_TITLE" ] && usage

SAFE_LABEL=$(echo "$DISC_LABEL" | tr '[:upper:]' '[:lower:]' | tr '_' '-' | tr '.' '-')
TIMESTAMP=$(date +%s)

echo "=== Pechka ETL Pipeline ==="
echo "Disc: $DISC_LABEL"
echo "Title: $CONTENT_TITLE"

# Phase 1: Extract (Bluray → NFS mkv/)
if [ "$SKIP_EXTRACT" = false ]; then
    echo ""
    echo "--- Phase 1: Extract ---"
    JOB_NAME="bluray-extract-${SAFE_LABEL}-${TIMESTAMP}"
    NODE_SELECTOR=""
    if [ -n "$BLURAY_NODE" ]; then
        NODE_SELECTOR="--overrides={\"spec\":{\"nodeName\":\"${BLURAY_NODE}\"}}"
    fi

    kubectl create job "$JOB_NAME" \
        --namespace "$NAMESPACE" \
        --from=job/bluray-extract \
        --image="${REGISTRY}/etl-extract:${IMAGE_TAG}" \
        $NODE_SELECTOR 2>/dev/null || \
    kubectl apply -f - <<EOF
$(sed -e "s/bluray-extract/${JOB_NAME}/g" \
      -e "s/image:.*/image: ${REGISTRY}\/etl-extract:${IMAGE_TAG}/" \
      k8s/etl/extract-job.yaml)
EOF

    echo "Waiting for extract job to complete..."
    kubectl wait job/"$JOB_NAME" --for=condition=complete --timeout=3600s --namespace "$NAMESPACE"
    echo "Extract complete."
fi

# Phase 2: Transform & Load (MKV → HLS → MinIO + PostgreSQL)
echo ""
echo "--- Phase 2: Transform & Load ---"

# Register content and get IDs via a helper pod
SHORT_ID=$(kubectl run "get-short-id-${TIMESTAMP}" \
    --namespace "$NAMESPACE" \
    --image="${REGISTRY}/etl-load:${IMAGE_TAG}" \
    --restart=Never --rm -i \
    --env="DISC_LABEL=${DISC_LABEL}" \
    --env="CONTENT_TITLE=${CONTENT_TITLE}" \
    --env="CONTENT_TYPE=video" \
    --command -- sh -c 'echo "register-only"' 2>/dev/null || true)

# Launch transcode jobs for each variant in parallel
VARIANTS=(original 1080p 720p 480p audio)
TRANSCODE_JOBS=()

for VARIANT in "${VARIANTS[@]}"; do
    JOB_NAME="transcode-${SAFE_LABEL}-${VARIANT}-${TIMESTAMP}"
    echo "Submitting transcode job: $JOB_NAME (variant=$VARIANT)"

    kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: ${JOB_NAME}
  namespace: ${NAMESPACE}
  labels:
    app: transcode
    disc: ${SAFE_LABEL}
    variant: ${VARIANT}
spec:
  ttlSecondsAfterFinished: 86400
  template:
    spec:
      containers:
        - name: ffmpeg
          image: ${REGISTRY}/etl-transform:${IMAGE_TAG}
          env:
            - name: INPUT_MKV
              value: /mnt/nfs/mkv/${DISC_LABEL}/title_00.mkv
            - name: OUTPUT_DIR
              value: /tmp/hls
            - name: SHORT_ID
              value: "${SHORT_ID:-PLACEHOLDER}"
            - name: VARIANT
              value: ${VARIANT}
            - name: CONTENT_ID
              value: "PLACEHOLDER"
            - name: MINIO_ALIAS
              value: minio
          volumeMounts:
            - name: nfs-mkv
              mountPath: /mnt/nfs/mkv
              readOnly: true
      volumes:
        - name: nfs-mkv
          nfs:
            server: "${NFS_SERVER:-nfs-server}"
            path: /mkv
      restartPolicy: Never
  backoffLimit: 2
EOF
    TRANSCODE_JOBS+=("$JOB_NAME")
done

echo "Waiting for all transcode jobs..."
for JOB in "${TRANSCODE_JOBS[@]}"; do
    kubectl wait job/"$JOB" --for=condition=complete --timeout=7200s --namespace "$NAMESPACE"
    echo "$JOB complete."
done

echo "All variants transcoded. Generating master playlist..."

# Generate master playlist
MASTER_JOB="generate-master-${SAFE_LABEL}-${TIMESTAMP}"
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: ${MASTER_JOB}
  namespace: ${NAMESPACE}
spec:
  ttlSecondsAfterFinished: 86400
  template:
    spec:
      containers:
        - name: generate-master
          image: ${REGISTRY}/etl-transform:${IMAGE_TAG}
          command: ["/usr/local/bin/generate_master.sh"]
          env:
            - name: SHORT_ID
              value: "${SHORT_ID:-PLACEHOLDER}"
            - name: CONTENT_ID
              value: "PLACEHOLDER"
            - name: OUTPUT_DIR
              value: /tmp/hls/master
            - name: MINIO_ALIAS
              value: minio
      restartPolicy: Never
  backoffLimit: 2
EOF

kubectl wait job/"$MASTER_JOB" --for=condition=complete --timeout=300s --namespace "$NAMESPACE"
echo "Master playlist generated."

# Phase 3: Thumbnail
echo ""
echo "--- Phase 3: Thumbnail ---"
THUMB_JOB="thumbnail-${SAFE_LABEL}-${TIMESTAMP}"
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: ${THUMB_JOB}
  namespace: ${NAMESPACE}
spec:
  ttlSecondsAfterFinished: 86400
  template:
    spec:
      containers:
        - name: thumbnail-analyzer
          image: ${REGISTRY}/etl-thumbnail:${IMAGE_TAG}
          args:
            - "--input"
            - "/mnt/nfs/mkv/${DISC_LABEL}/title_00.mkv"
            - "--short-id"
            - "${SHORT_ID:-PLACEHOLDER}"
            - "--content-id"
            - "PLACEHOLDER"
      restartPolicy: Never
  backoffLimit: 2
EOF

kubectl wait job/"$THUMB_JOB" --for=condition=complete --timeout=1800s --namespace "$NAMESPACE"
echo "Thumbnails generated."

echo ""
echo "=== Pipeline complete! ==="
echo "Content short_id: ${SHORT_ID:-PLACEHOLDER}"
