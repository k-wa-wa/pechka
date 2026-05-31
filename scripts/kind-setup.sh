#!/usr/bin/env bash
# Create Kind cluster and deploy local overlay
set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-pechka-cluster}"

echo "=== Creating Kind Cluster ==="
if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "Cluster '${CLUSTER_NAME}' already exists, skipping creation."
else
    kind create cluster --name "${CLUSTER_NAME}"
fi

echo ""
echo "=== Deploying to Kind ==="
kubectl apply -f k8s/base/namespace.yaml
# Wait for the default service account to be created before applying the overlay
kubectl wait serviceaccount/default -n pechka --for=jsonpath='{.metadata.name}'=default --timeout=30s
kubectl apply -k k8s/overlays/local

echo ""
echo "=== Kind setup complete! ==="
echo "Run 'scripts/build-and-load.sh' to build and load container images."
