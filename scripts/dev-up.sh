#!/usr/bin/env bash
# Bootstrap local development environment end-to-end:
#   1. Create Kind cluster and deploy Kubernetes manifests
#   2. Build all container images and load into Kind
#   3. Print access instructions
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${REPO_ROOT}"

bash scripts/kind-setup.sh
bash scripts/build-and-load.sh

echo ""
echo "=== Local environment is ready! ==="
echo ""
echo "Start port-forwarding:"
echo "  kubectl port-forward svc/nginx -n pechka 8000:80"
echo ""
echo "Then open http://localhost:8000 in your browser."
echo ""
echo "Verify with:"
echo "  curl -s http://localhost:8000/api/v1/contents"
