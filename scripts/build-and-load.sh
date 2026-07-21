#!/usr/bin/env bash
# Build all container images and load them into the Kind cluster
set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-pechka-cluster}"
REGISTRY="ghcr.io/k-wa-wa/pechka"
ETL_COMPONENTS=(extract transform load generate-thumbnail refresh-latest-playlist)

# Use system Docker, not Docker Desktop socket
# unset DOCKER_HOST


echo "=== Building API Image ==="
docker build -t "${REGISTRY}-api:latest" ./api

echo ""
echo "=== Building Frontend Image ==="
docker build -t "${REGISTRY}-frontend:latest" ./frontend

echo ""
echo "=== Building ETL Images ==="
for COMP in "${ETL_COMPONENTS[@]}"; do
    echo "Building ${REGISTRY}/etl-${COMP}:latest..."
    if [ "$COMP" = "extract" ] || [ "$COMP" = "transform" ] || [ "$COMP" = "load" ]; then
        docker build -t "${REGISTRY}/etl-${COMP}:latest" -f "./batch/etl/Dockerfile.${COMP}" .
    else
        docker build -t "${REGISTRY}/etl-${COMP}:latest" "./batch/${COMP}"
    fi
done

echo ""
echo "=== Loading Images into Kind Cluster (${CLUSTER_NAME}) ==="
kind load docker-image \
    "${REGISTRY}-api:latest" \
    "${REGISTRY}-frontend:latest" \
    --name "${CLUSTER_NAME}"

for COMP in "${ETL_COMPONENTS[@]}"; do
    echo "Loading ${REGISTRY}/etl-${COMP}:latest..."
    kind load docker-image "${REGISTRY}/etl-${COMP}:latest" --name "${CLUSTER_NAME}"
done

echo ""
echo "=== Restarting deployments to pick up new images ==="
kubectl rollout restart deployment/api deployment/frontend -n pechka
kubectl rollout status deployment/api deployment/frontend -n pechka

echo ""
echo "=== Build and load complete! ==="
