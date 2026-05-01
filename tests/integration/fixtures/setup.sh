#!/bin/sh
# Generate minimal HLS fixture files for integration testing.
# These simulate the output of the transcode step.

set -e

DISC_LABEL="${1:-TEST_DISC_001}"
OUT_DIR="$(dirname "$0")/hls/${DISC_LABEL}"

mkdir -p "$OUT_DIR"

# Minimal valid HLS playlist
cat > "$OUT_DIR/720p.m3u8" << 'EOF'
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:10.000000,
720p_0000.ts
#EXT-X-ENDLIST
EOF

cat > "$OUT_DIR/480p.m3u8" << 'EOF'
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:10.000000,
480p_0000.ts
#EXT-X-ENDLIST
EOF

cat > "$OUT_DIR/audio.m3u8" << 'EOF'
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:10.000000,
audio_0000.ts
#EXT-X-ENDLIST
EOF

cat > "$OUT_DIR/master.m3u8" << 'EOF'
#EXTM3U
#EXT-X-VERSION:3

#EXT-X-STREAM-INF:BANDWIDTH=3128000,RESOLUTION=1280x720,CODECS="avc1.4d001f,mp4a.40.2"
720p.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=1628000,RESOLUTION=854x480,CODECS="avc1.42e01f,mp4a.40.2"
480p.m3u8

#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",NAME="Audio Only",DEFAULT=NO,URI="audio.m3u8"
EOF

# Minimal MPEG-TS segments (188-byte null TS packet repeated)
# This is not a valid video, but it's enough for the upload test
python3 -c "
import sys
# Minimal TS sync byte sequence (0x47 = sync byte)
pkt = bytes([0x47, 0x40, 0x00, 0x10]) + bytes(184)  # 188 bytes
sys.stdout.buffer.write(pkt * 10)
" > "$OUT_DIR/720p_0000.ts"

cp "$OUT_DIR/720p_0000.ts" "$OUT_DIR/480p_0000.ts"
cp "$OUT_DIR/720p_0000.ts" "$OUT_DIR/audio_0000.ts"

echo "Fixtures created in $OUT_DIR:"
ls -lh "$OUT_DIR"
