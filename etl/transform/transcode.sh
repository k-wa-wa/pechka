#!/bin/bash
set -euo pipefail

INPUT_MKV="${INPUT_MKV:?INPUT_MKV is required}"
OUTPUT_DIR="${OUTPUT_DIR:-/tmp/hls}"
SHORT_ID="${SHORT_ID:?SHORT_ID is required}"
VARIANT="${VARIANT:?VARIANT is required}"
MINIO_ALIAS="${MINIO_ALIAS:-minio}"
MINIO_BUCKET="${MINIO_BUCKET:?MINIO_BUCKET is required}"
POSTGRES_DSN="${POSTGRES_DSN:?POSTGRES_DSN is required}"
CONTENT_ID="${CONTENT_ID:?CONTENT_ID is required}"

mkdir -p "$OUTPUT_DIR"

echo "Transcoding $INPUT_MKV as variant=$VARIANT short_id=$SHORT_ID"

case "$VARIANT" in
    original)
        ffmpeg -i "$INPUT_MKV" \
            -c:v copy -c:a aac -b:a 192k \
            -f hls -hls_time 6 -hls_list_size 0 \
            -hls_segment_filename "${OUTPUT_DIR}/original_%04d.ts" \
            "${OUTPUT_DIR}/original.m3u8"
        BANDWIDTH=""
        RESOLUTION=""
        CODECS="avc1.640028,mp4a.40.2"
        HLS_KEY="resources/hls/${SHORT_ID}/original.m3u8"
        ;;
    1080p)
        ffmpeg -i "$INPUT_MKV" \
            -vf scale=1920:1080 -c:v libx264 -preset fast \
            -b:v 6000k -maxrate 6500k -bufsize 12000k \
            -c:a aac -b:a 192k \
            -f hls -hls_time 6 -hls_list_size 0 \
            -hls_segment_filename "${OUTPUT_DIR}/1080p_%04d.ts" \
            "${OUTPUT_DIR}/1080p.m3u8"
        BANDWIDTH=6192000
        RESOLUTION="1920x1080"
        CODECS="avc1.640028,mp4a.40.2"
        HLS_KEY="resources/hls/${SHORT_ID}/1080p.m3u8"
        ;;
    720p)
        ffmpeg -i "$INPUT_MKV" \
            -vf scale=1280:720 -c:v libx264 -preset fast \
            -b:v 3000k -maxrate 3500k -bufsize 6000k \
            -c:a aac -b:a 128k \
            -f hls -hls_time 6 -hls_list_size 0 \
            -hls_segment_filename "${OUTPUT_DIR}/720p_%04d.ts" \
            "${OUTPUT_DIR}/720p.m3u8"
        BANDWIDTH=3128000
        RESOLUTION="1280x720"
        CODECS="avc1.4d001f,mp4a.40.2"
        HLS_KEY="resources/hls/${SHORT_ID}/720p.m3u8"
        ;;
    480p)
        ffmpeg -i "$INPUT_MKV" \
            -vf scale=854:480 -c:v libx264 -preset fast \
            -b:v 1500k -maxrate 2000k -bufsize 3000k \
            -c:a aac -b:a 128k \
            -f hls -hls_time 6 -hls_list_size 0 \
            -hls_segment_filename "${OUTPUT_DIR}/480p_%04d.ts" \
            "${OUTPUT_DIR}/480p.m3u8"
        BANDWIDTH=1628000
        RESOLUTION="854x480"
        CODECS="avc1.42e01f,mp4a.40.2"
        HLS_KEY="resources/hls/${SHORT_ID}/480p.m3u8"
        ;;
    audio)
        ffmpeg -i "$INPUT_MKV" \
            -vn -c:a aac -b:a 192k \
            -f hls -hls_time 6 -hls_list_size 0 \
            -hls_segment_filename "${OUTPUT_DIR}/audio_%04d.ts" \
            "${OUTPUT_DIR}/audio.m3u8"
        BANDWIDTH=192000
        RESOLUTION=""
        CODECS="mp4a.40.2"
        HLS_KEY="resources/hls/${SHORT_ID}/audio.m3u8"
        ;;
    *)
        echo "ERROR: Unknown variant: $VARIANT"
        exit 1
        ;;
esac

echo "Transcode complete. Uploading to MinIO..."
mc cp --recursive "${OUTPUT_DIR}/" "${MINIO_ALIAS}/${MINIO_BUCKET}/resources/hls/${SHORT_ID}/"

echo "Upload complete. Registering variant in PostgreSQL..."

BANDWIDTH_VAL="NULL"
RESOLUTION_VAL="NULL"
CODECS_VAL="NULL"
[ -n "$BANDWIDTH" ] && BANDWIDTH_VAL="$BANDWIDTH"
[ -n "$RESOLUTION" ] && RESOLUTION_VAL="'$RESOLUTION'"
[ -n "$CODECS" ] && CODECS_VAL="'$CODECS'"

psql "$POSTGRES_DSN" -c "
    INSERT INTO video_variants (content_id, variant_type, hls_key, bandwidth, resolution, codecs)
    VALUES (
        '${CONTENT_ID}',
        '${VARIANT}',
        '${HLS_KEY}',
        ${BANDWIDTH_VAL},
        ${RESOLUTION_VAL},
        ${CODECS_VAL}
    );
"

echo "Variant $VARIANT registered successfully."
