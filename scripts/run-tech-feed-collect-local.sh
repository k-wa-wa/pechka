#!/usr/bin/env bash
set -euo pipefail

# pechka batch-tech-feed collect ローカル実行用スクリプト
#
# このスクリプトは k8s マニフェストから各種設定値を抽出し、
# SOPS で暗号化された Secret を復号して環境変数に設定した上で、
# go run ./batch-tech-feed/collect を実行する。

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

CONFIGMAP_FILE="k8s/base/infra/configmap/configmap.yaml"
PROD_SECRETS_FILE="k8s/overlays/prod/secrets/prod-secrets.yaml"

if [ ! -f "$CONFIGMAP_FILE" ]; then
  echo "Error: ConfigMap ファイル ($CONFIGMAP_FILE) が見つかりません。" >&2
  exit 1
fi

if [ ! -f "$PROD_SECRETS_FILE" ]; then
  echo "Error: Secret ファイル ($PROD_SECRETS_FILE) が見つかりません。" >&2
  exit 1
fi

# 1. SOPS を用いた Secret 復元
if ! command -v sops &> /dev/null; then
  echo "Error: sops コマンドがインストールされていません。" >&2
  echo "brew install sops 等で sops をインストールしてください。" >&2
  exit 1
fi

echo "=== SOPS から Secret を復元中 ==="
OPENAI_API_KEY=$(sops -d --extract '["OPENAI_API_KEY"]' "$PROD_SECRETS_FILE" 2>/dev/null || \
                 sops -d "$PROD_SECRETS_FILE" | grep -E "^\s*OPENAI_API_KEY:" | awk '{print $2}')
export OPENAI_API_KEY

if [ -z "$OPENAI_API_KEY" ]; then
  echo "Error: OPENAI_API_KEY の取得に失敗しました。" >&2
  exit 1
fi
echo "OPENAI_API_KEY の復元に成功した。"

# 2. k8s ConfigMap (yaml) から設定値の取得
get_config_val() {
  local key="$1"
  grep -E "^\s*${key}:" "$CONFIGMAP_FILE" | head -n 1 | sed -E 's/^[[:space:]]*[a-zA-Z0-9_-]+:[[:space:]]*"?([^"]*)"?/\1/'
}

export OPENAI_BASE_URL="$(get_config_val "openai-base-url")"
export OPENAI_MODEL="$(get_config_val "openai-model")"
export BARE_WEB_PROXY_URL="${BARE_WEB_PROXY_URL:-https://bwproxy.cluster.wpc/}"
export SKIP_TLS_VERIFY="${SKIP_TLS_VERIFY:-true}"

echo "=== K8s ConfigMap および固定値から取得した設定 ==="
echo "  OPENAI_BASE_URL: $OPENAI_BASE_URL"
echo "  OPENAI_MODEL: $OPENAI_MODEL"
echo "  BARE_WEB_PROXY_URL: $BARE_WEB_PROXY_URL"
echo "  SKIP_TLS_VERIFY: $SKIP_TLS_VERIFY"

# 3. コマンド引数の解析
MODE="${1:-pipeline}"   # サブコマンド (pipeline / collect / filter / enrich / compose)
SOURCE_TYPE="${2:-ai}"  # ソース種別 (ai / k8s)
OUTPUT_FILE="${3:-/tmp/script.json}"

SOURCES_FILE="$REPO_ROOT/k8s/base/tech-feed/sources-${SOURCE_TYPE}.json"
PROMPT_FILE="$REPO_ROOT/k8s/base/tech-feed/prompt-${SOURCE_TYPE}.txt"

if [ ! -f "$SOURCES_FILE" ]; then
  echo "Error: ソース定義ファイル ($SOURCES_FILE) が見つかりません。" >&2
  exit 1
fi

echo "=== batch-tech-feed/collect を実行中 ==="
echo "  Mode: $MODE"
echo "  Sources: $SOURCES_FILE"
echo "  Prompt: $PROMPT_FILE"
echo "  Output: $OUTPUT_FILE"
echo "----------------------------------------"

# 4. batch-tech-feed/collect ディレクトリで go run を実行
(
  cd "$REPO_ROOT/batch-tech-feed/collect"
  go run . \
    "$MODE" \
    -sources "$SOURCES_FILE" \
    -prompt "$PROMPT_FILE" \
    -since-days 2 \
    -topics 3 \
    -output "$OUTPUT_FILE"
)

echo "----------------------------------------"
echo "完了。成果物は $OUTPUT_FILE に出力された。"
