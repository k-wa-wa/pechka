#!/usr/bin/env bash
# Build and load all ETL images to Kind
set -euo pipefail

CLUSTER_NAME="pechka-cluster"
REGISTRY="ghcr.io/k-wa-wa/pechka"
COMPONENTS=(extract transform load thumbnail refresh-latest-playlist)

# Ensure system Docker is used
unset DOCKER_HOST

echo "=== Building ETL Images ==="
for COMP in "${COMPONENTS[@]}"; do
    echo "Building ${REGISTRY}/etl-${COMP}:latest..."
    docker build -t "${REGISTRY}/etl-${COMP}:latest" "./etl/${COMP}"
done

echo ""
echo "=== Loading Images into Kind Cluster (${CLUSTER_NAME}) ==="
for COMP in "${COMPONENTS[@]}"; do
    echo "Loading ${REGISTRY}/etl-${COMP}:latest..."
    kind load docker-image "${REGISTRY}/etl-${COMP}:latest" --name "${CLUSTER_NAME}"
done

echo ""
echo "=== Build and load complete! ==="
