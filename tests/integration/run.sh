#!/bin/sh
# ETL TL (Transform/Load) integration test runner.
# Starts Docker Compose, runs load component against test PostgreSQL + MinIO,
# and verifies results. Cleans up on exit.

set -e
cd "$(dirname "$0")"

# Generate fixture HLS files if not already present
if [ ! -d fixtures/hls/TEST_DISC_001 ]; then
  echo "Generating fixture HLS files..."
  sh fixtures/setup.sh
fi

cleanup() {
  echo "Cleaning up..."
  docker compose -f docker-compose.test.yml down -v --remove-orphans 2>/dev/null || true
}
trap cleanup EXIT

echo "=== ETL Integration Tests ==="

# Apply DB migrations before starting test containers
echo "[1/4] Starting infrastructure..."
docker compose -f docker-compose.test.yml up -d postgres minio
docker compose -f docker-compose.test.yml up minio-init

echo "[2/4] Applying DB migrations..."
docker compose -f docker-compose.test.yml run --rm \
  -e PGPASSWORD=testpass \
  postgres sh -c "
    until pg_isready -h postgres -U postgres; do sleep 1; done
    psql -h postgres -U postgres -d pechka_test -f /dev/stdin
  " < ../../db/migrations/001_create_tables.up.sql 2>/dev/null || \
  docker exec "$(docker compose -f docker-compose.test.yml ps -q postgres)" \
    sh -c "PGPASSWORD=testpass psql -U postgres -d pechka_test" \
    < ../../db/migrations/001_create_tables.up.sql

echo "[3/4] Running load integration test..."
docker compose -f docker-compose.test.yml run --rm load-integration-test

echo "[4/4] Verifying results..."
docker compose -f docker-compose.test.yml run --rm verify

echo ""
echo "=== All integration tests passed ==="
