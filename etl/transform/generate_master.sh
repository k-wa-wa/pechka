#!/bin/bash
set -euo pipefail

SHORT_ID="${SHORT_ID:?SHORT_ID is required}"
CONTENT_ID="${CONTENT_ID:?CONTENT_ID is required}"
OUTPUT_DIR="${OUTPUT_DIR:-/tmp/hls/master}"
MINIO_ALIAS="${MINIO_ALIAS:-minio}"
MINIO_BUCKET="${MINIO_BUCKET:?MINIO_BUCKET is required}"
POSTGRES_DSN="${POSTGRES_DSN:?POSTGRES_DSN is required}"

mkdir -p "$OUTPUT_DIR"

cat > "${OUTPUT_DIR}/master.m3u8" <<'EOF'
#EXTM3U
#EXT-X-VERSION:3

#EXT-X-STREAM-INF:BANDWIDTH=0,CODECS="avc1.640028,mp4a.40.2"
original.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=6192000,RESOLUTION=1920x1080,CODECS="avc1.640028,mp4a.40.2"
1080p.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=3128000,RESOLUTION=1280x720,CODECS="avc1.4d001f,mp4a.40.2"
720p.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=1628000,RESOLUTION=854x480,CODECS="avc1.42e01f,mp4a.40.2"
480p.m3u8

#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",NAME="Audio Only",DEFAULT=NO,URI="audio.m3u8"
EOF

mc cp "${OUTPUT_DIR}/master.m3u8" "${MINIO_ALIAS}/${MINIO_BUCKET}/resources/hls/${SHORT_ID}/master.m3u8"

psql "$POSTGRES_DSN" -c "
    INSERT INTO video_variants (content_id, variant_type, hls_key)
    VALUES (
        '${CONTENT_ID}',
        'master',
        'resources/hls/${SHORT_ID}/master.m3u8'
    );

    UPDATE contents SET status = 'ready', published_at = NOW()
    WHERE id = '${CONTENT_ID}';
"

echo "Master playlist generated and content marked as ready."
