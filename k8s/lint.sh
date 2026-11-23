#!/usr/bin/env nix-shell
#!nix-shell -i bash -p benthos kustomize kubeconform pluto conftest jq yamllint python3Packages.pyyaml postgresql_16 -I nixpkgs=channel:nixpkgs-unstable

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

ONLY_PECHKA=false
for arg in "$@"; do
  case "$arg" in
    --only-pechka)
      ONLY_PECHKA=true
      ;;
    -h|--help)
      echo "Usage: $0 [--only-pechka]"
      exit 0
      ;;
    *)
      echo "❌ Unknown option: $arg" >&2
      echo "Usage: $0 [--only-pechka]" >&2
      exit 1
      ;;
  esac
done

TMP_DIR=""
cleanup() {
  if [ -n "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
  fi
}
trap cleanup EXIT

run_common_validation() {
  local local_runner="${HOME}/ghq/github.com/k-wa-wa/nuage-workspace/.github/workflows/validate/k8s/run.sh"
  local runner
  if [ -x "$local_runner" ]; then
    runner="$local_runner"
  else
    TMP_DIR="$(mktemp -d)"
    curl -sSfL "https://github.com/k-wa-wa/nuage-workspace/archive/refs/heads/master.tar.gz" \
      | tar -xz -C "$TMP_DIR" "nuage-workspace-master/.github/workflows/validate/k8s"
    runner="${TMP_DIR}/nuage-workspace-master/.github/workflows/validate/k8s/run.sh"
  fi

  echo "=== Running nuage-workspace common K8s validation ==="
  if ! "$runner" --yamllint-path "k8s" --prod "k8s/overlays/prod"; then
    FAILED=1
  fi
  if ! "$runner" --preview "k8s/overlays/preview"; then
    FAILED=1
  fi
  echo ""
}

run_pechka_specific_checks() {
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
  local prod_manifests_file
  prod_manifests_file="$(mktemp --suffix=.yaml)"
  if kustomize build "k8s/overlays/prod" > "$prod_manifests_file"; then
    if grep -qF 's3-fallback-endpoint: minio:9000' "$prod_manifests_file"; then
      echo "❌ prod pechka-config has unresolved base fallback endpoint 'minio:9000' (minio service does not exist in prod)!"
      FAILED=1
    fi
    if grep -qF 's3-endpoint: minio:9000' "$prod_manifests_file"; then
      echo "❌ prod pechka-config has unresolved base endpoint 'minio:9000'!"
      FAILED=1
    fi
  else
    echo "Kustomize build failed for k8s/overlays/prod!"
    FAILED=1
  fi
  rm -f "$prod_manifests_file"
  echo ""

  local preview_secret="k8s/overlays/preview-secret"
  echo "=== Running pechka-specific checks for ${preview_secret} ==="
  if [ -d "${preview_secret}" ]; then
    local secret_manifests_file
    secret_manifests_file="$(mktemp --suffix=.yaml)"
    if kustomize build "${preview_secret}" > "$secret_manifests_file"; then
      echo "--- kubeconform (${preview_secret}) ---"
      if ! kubeconform -summary -strict -ignore-missing-schemas \
        -schema-location default \
        -schema-location 'https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json' \
        "$secret_manifests_file"; then
        echo "kubeconform failed for ${preview_secret}!"
        FAILED=1
      fi

      echo "--- pluto (${preview_secret}) ---"
      if ! pluto detect "$secret_manifests_file"; then
        echo "pluto failed for ${preview_secret}!"
        FAILED=1
      fi
    else
      echo "Kustomize build failed for ${preview_secret}!"
      FAILED=1
    fi
    rm -f "$secret_manifests_file"
  else
    echo "Directory ${preview_secret} does not exist. Skipping."
  fi
  echo ""
}

FAILED=0

if [ "$ONLY_PECHKA" != "true" ]; then
  run_common_validation
fi

run_pechka_specific_checks

if [ "$FAILED" -ne 0 ]; then
  echo "❌ Static analysis failed!"
  exit 1
else
  echo "✅ All static analysis passed successfully!"
  exit 0
fi
