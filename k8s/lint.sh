#!/usr/bin/env nix-shell
#!nix-shell -i bash -p benthos kustomize kubeconform pluto conftest jq yamllint python3Packages.pyyaml postgresql_16 -I nixpkgs=channel:nixpkgs-unstable

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

TMP_DIR=""
cleanup() {
  if [ -n "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
  fi
}
trap cleanup EXIT

LOCAL_RUNNER="${HOME}/ghq/github.com/k-wa-wa/nuage-workspace/.github/workflows/validate/k8s/run.sh"
if [ -x "$LOCAL_RUNNER" ]; then
  RUNNER="$LOCAL_RUNNER"
else
  TMP_DIR="$(mktemp -d)"
  curl -sSfL "https://github.com/k-wa-wa/nuage-workspace/archive/refs/heads/master.tar.gz" \
    | tar -xz -C "$TMP_DIR" "nuage-workspace-master/.github/workflows/validate/k8s"
  RUNNER="${TMP_DIR}/nuage-workspace-master/.github/workflows/validate/k8s/run.sh"
fi

FAILED=0

echo "=== Running nuage-workspace common K8s validation ==="
if ! "$RUNNER" \
  --yamllint-path "k8s" \
  --prod-overlays "k8s/overlays/prod" \
  --preview-overlays "k8s/overlays/preview" \
  --fail-on-policy "true"; then
  FAILED=1
fi
echo ""

echo "=== Checking \$(VAR) references and PreSync dependencies ==="
if python3 scripts/check-k8s-env-vars.py; then
  echo "env vars check passed!"
else
  echo "env vars check failed!"
  FAILED=1
fi
echo ""

echo "=== Testing Benthos stream Bloblang mappings and switch branches ==="
if python3 scripts/test-benthos-streams.py; then
  echo "Benthos stream tests passed!"
else
  echo "Benthos stream tests failed!"
  FAILED=1
fi
echo ""

echo "=== Testing PostgreSQL migrations ==="
if python3 scripts/test-postgres-migrations.py; then
  echo "PostgreSQL migration tests passed!"
else
  echo "PostgreSQL migration tests failed!"
  FAILED=1
fi
echo ""

echo "=== Validating prod S3 endpoint overrides ==="
PROD_MANIFESTS_FILE="$(mktemp --suffix=.yaml)"
if kustomize build "k8s/overlays/prod" > "$PROD_MANIFESTS_FILE"; then
  if grep -qF 's3-fallback-endpoint: minio:9000' "$PROD_MANIFESTS_FILE"; then
    echo "❌ prod pechka-config has unresolved base fallback endpoint 'minio:9000' (minio service does not exist in prod)!"
    FAILED=1
  fi
  if grep -qF 's3-endpoint: minio:9000' "$PROD_MANIFESTS_FILE"; then
    echo "❌ prod pechka-config has unresolved base endpoint 'minio:9000'!"
    FAILED=1
  fi
else
  echo "Kustomize build failed for k8s/overlays/prod!"
  FAILED=1
fi
rm -f "$PROD_MANIFESTS_FILE"
echo ""

PREVIEW_SECRET="k8s/overlays/preview-secret"
echo "=== Running pechka-specific checks for ${PREVIEW_SECRET} ==="
if [ -d "${PREVIEW_SECRET}" ]; then
  SECRET_MANIFESTS_FILE="$(mktemp --suffix=.yaml)"
  if kustomize build "${PREVIEW_SECRET}" > "$SECRET_MANIFESTS_FILE"; then
    echo "--- kubeconform (${PREVIEW_SECRET}) ---"
    if ! kubeconform -summary -strict -ignore-missing-schemas \
      -schema-location default \
      -schema-location 'https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json' \
      "$SECRET_MANIFESTS_FILE"; then
      echo "kubeconform failed for ${PREVIEW_SECRET}!"
      FAILED=1
    fi

    echo "--- pluto (${PREVIEW_SECRET}) ---"
    if ! pluto detect "$SECRET_MANIFESTS_FILE"; then
      echo "pluto failed for ${PREVIEW_SECRET}!"
      FAILED=1
    fi
  else
    echo "Kustomize build failed for ${PREVIEW_SECRET}!"
    FAILED=1
  fi
  rm -f "$SECRET_MANIFESTS_FILE"
else
  echo "Directory ${PREVIEW_SECRET} does not exist. Skipping."
fi
echo ""

if [ "$FAILED" -ne 0 ]; then
  echo "❌ Static analysis failed!"
  exit 1
else
  echo "✅ All static analysis passed successfully!"
  exit 0
fi
