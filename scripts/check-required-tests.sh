#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
FAILED=0

echo "=== Checking required test files across repository ==="

# 1. Go modules: every package directory with non-test .go files (excluding pure main entrypoints or pure type declarations) must have at least one *_test.go file
GO_MODULES=("api" "batch/etl" "batch/refresh-latest-playlist" "batch-tech-feed/collect" "batch-tech-feed/produce")

for mod in "${GO_MODULES[@]}"; do
  while IFS= read -r -d '' dir; do
    # Count non-test .go files
    go_files=$(find "$dir" -maxdepth 1 -name "*.go" ! -name "*_test.go" | wc -l | tr -d ' ')
    if [ "$go_files" -eq 0 ]; then
      continue
    fi
    test_files=$(find "$dir" -maxdepth 1 -name "*_test.go" | wc -l | tr -d ' ')
    rel_dir="${dir#$ROOT_DIR/}"
    if [ "$test_files" -eq 0 ]; then
      echo "❌ Missing Go test file (*_test.go) in package directory: $rel_dir"
      FAILED=1
    fi
  done < <(find "$ROOT_DIR/$mod" -type d -not -path "*/.*" -not -path "*/remotion*" -not -path "*/examples*" -print0)
done

# 2. Python batches: every main .py file must have a corresponding test_*.py file
PY_DIRS=("batch/generate-thumbnail" "batch/subtitle")
for pydir in "${PY_DIRS[@]}"; do
  for pyfile in "$ROOT_DIR/$pydir"/*.py; do
    base=$(basename "$pyfile")
    if [[ "$base" == test_* ]] || [[ "$base" == "__init__.py" ]]; then
      continue
    fi
    expected="$ROOT_DIR/$pydir/test_${base}"
    if [ ! -f "$expected" ]; then
      echo "❌ Missing Python test file: $pydir/test_${base} (for $pydir/$base)"
      FAILED=1
    fi
  done
done

# 3. Frontend: run frontend/scripts/check-tests.mjs if present
if [ -f "$ROOT_DIR/frontend/scripts/check-tests.mjs" ]; then
  node "$ROOT_DIR/frontend/scripts/check-tests.mjs" || FAILED=1
fi

if [ "$FAILED" -ne 0 ]; then
  echo "❌ Required test file check failed!"
  exit 1
fi

echo "✅ All required test files are present."
