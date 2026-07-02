#!/usr/bin/env nix-shell
#!nix-shell -i bash -p kustomize kubeconform pluto kube-linter yamllint -I nixpkgs=channel:nixpkgs-unstable

set -euo pipefail

# エラー発生時のフラグ
FAILED=0

echo "=== 1. Running yamllint ==="
# yamllint はプロジェクトルートの .yamllint 設定ファイルを自動的に参照します
if yamllint k8s; then
  echo "yamllint passed successfully!"
else
  echo "yamllint failed!"
  FAILED=1
fi
echo ""

# 各オーバーレイ（環境）に対するマニフェストチェック
OVERLAYS=("k8s/overlays/local" "k8s/overlays/prod")

for OVERLAY in "${OVERLAYS[@]}"; do
  echo "=== Running Manifest Checks for ${OVERLAY} ==="
  if [ ! -d "${OVERLAY}" ]; then
    echo "Directory ${OVERLAY} does not exist. Skipping."
    continue
  fi

  # Kustomize ビルド
  echo "Building manifests with kustomize..."
  if ! MANIFESTS=$(kustomize build "${OVERLAY}"); then
    echo "Kustomize build failed for ${OVERLAY}!"
    FAILED=1
    continue
  fi

  # kubeconform (スキーマチェック)
  echo "Validating with kubeconform..."
  # Argo Workflowsなどのカスタムリソース定義(CRD)に対応するため、外部のCRDカタログも参照します。
  if echo "$MANIFESTS" | kubeconform -summary -strict -ignore-missing-schemas \
    -schema-location default \
    -schema-location 'https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json'; then
    echo "kubeconform passed!"
  else
    echo "kubeconform failed!"
    FAILED=1
  fi

  # pluto (非推奨/廃止APIチェック)
  echo "Checking for deprecated APIs with pluto..."
  if echo "$MANIFESTS" | pluto detect -; then
    echo "pluto passed!"
  else
    echo "pluto failed!"
    FAILED=1
  fi

  # kube-linter (ベストプラクティスチェック)
  # デフォルトでは警告が多いため、警告の出力は行いますが、このエラーによるスクリプト全体の失敗は無視します。
  echo "Checking best practices with kube-linter (warning only)..."
  if echo "$MANIFESTS" | kube-linter lint -; then
    echo "kube-linter passed!"
  else
    echo "kube-linter found some recommendations."
  fi
  echo ""
done

if [ "$FAILED" -ne 0 ]; then
  echo "❌ Static analysis failed!"
  exit 1
else
  echo "✅ All static analysis passed successfully!"
  exit 0
fi
