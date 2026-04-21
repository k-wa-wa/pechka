#!/bin/bash
set -euo pipefail

DEVICE="${DEVICE:-/dev/sr0}"
NFS_MKV_DIR="${NFS_MKV_DIR:-/mnt/nfs/mkv}"
POSTGRES_DSN="${POSTGRES_DSN:?POSTGRES_DSN is required}"

# Get disc label
DISC_LABEL=$(blkid "$DEVICE" | grep -oP 'LABEL="\K[^"]+' || true)
if [ -z "$DISC_LABEL" ]; then
    DISC_LABEL="DISC_$(date +%Y%m%d_%H%M%S)"
    echo "WARNING: Could not get disc label, using generated: $DISC_LABEL"
fi

echo "Extracting disc: $DISC_LABEL"
OUTPUT_DIR="${NFS_MKV_DIR}/${DISC_LABEL}"
mkdir -p "$OUTPUT_DIR"

# Extract all titles using MakeMKV
makemkvcon mkv "disc:0" all "$OUTPUT_DIR"

echo "Extraction complete. Files in $OUTPUT_DIR:"
ls -lh "$OUTPUT_DIR"

# Register disc in PostgreSQL
psql "$POSTGRES_DSN" -c "
    INSERT INTO discs (label)
    VALUES ('${DISC_LABEL}')
    ON CONFLICT (label) DO NOTHING;
"

echo "Disc $DISC_LABEL registered in database."
