#!/usr/bin/env bash
set -euo pipefail

# Usage: scripts/check-go-coverage.sh <module_dir> [threshold_percent]
MOD_DIR="${1:?Module directory required}"
THRESHOLD="${2:-80.0}"

cd "$MOD_DIR"

echo "=== Running Go tests with coverage in $MOD_DIR (threshold: ${THRESHOLD}%) ==="
CGO_ENABLED=0 go test -race -coverprofile=coverage.out -covermode=atomic ./...

TOTAL_COV=$(go tool cover -func=coverage.out | grep '^total:' | awk '{print substr($3, 1, length($3)-1)}')
echo "Total statement coverage in $MOD_DIR: ${TOTAL_COV}% (required: >= ${THRESHOLD}%)"

awk -v total="$TOTAL_COV" -v thresh="$THRESHOLD" 'BEGIN {
  if (total + 0 < thresh + 0) {
    printf "❌ Coverage %.1f%% is below threshold %.1f%% in %s\n", total, thresh, "'"$MOD_DIR"'"
    exit 1
  }
  printf "✅ Coverage %.1f%% meets threshold %.1f%% in %s\n", total, thresh, "'"$MOD_DIR"'"
}'
