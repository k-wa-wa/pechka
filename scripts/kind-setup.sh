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
kubectl create namespace argo --dry-run=client -o yaml | kubectl apply -f -
# Wait for the default service account to be created before applying the overlay

until kubectl get serviceaccount/default -n pechka >/dev/null 2>&1; do
    echo "Waiting for default serviceaccount in namespace pechka..."
    sleep 1
done
kubectl apply -k k8s/overlays/local


echo ""
echo "=== Kind setup complete! ==="
echo "Run 'scripts/build-and-load.sh' to build and load container images."
