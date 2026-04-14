# Bluray ディスク取り込みパイプライン設計

## 1. 概要

Bluray ディスクから動画コンテンツを取り込み、HLS ABR 配信可能な状態にするまでのパイプライン設計。
物理デバイスへの依存を最小化するため、パイプラインを2つのフェーズに分割する。

```
Phase 1（物理デバイス依存）: Bluray Disc → MakeMKV → NFS /mkv/
                                                           ↓（永続保持）
Phase 2（ファイル操作のみ）: NFS /mkv/ → ffmpeg ABR → MinIO /hls/ + PostgreSQL
```

## 2. Phase 1: Extract（Bluray → MKV）

### 2.1 概要

- 物理 Bluray ドライブに接続したノード上で実行。
- MakeMKV を使用してディスク内の全タイトルを MKV ファイルとして抽出。
- 抽出した MKV は NFS の `/mnt/nfs/mkv/{DISC_LABEL}/` に保存。
- 再変換を可能にするため、MKV は **永続保持**（削除しない）。

### 2.2 実行環境

- K8s Job（`nodeName` または `nodeAffinity` で Bluray ドライブ搭載ノードに固定）
- ホストのデバイス (`/dev/sr0`) を Pod にマウント
- MakeMKV コンテナイメージ（makemkv/makemkv-oss ベース）

### 2.3 処理フロー

```bash
# ディスクラベル取得
DISC_LABEL=$(blkid /dev/sr0 | grep -oP 'LABEL="\K[^"]+')

# MakeMKV で全タイトルを抽出
makemkvcon mkv disc:0 all /mnt/nfs/mkv/${DISC_LABEL}/

# 完了後、discs テーブルに登録
psql -c "INSERT INTO discs (label) VALUES ('${DISC_LABEL}') ON CONFLICT DO NOTHING"
```

### 2.4 出力

```
/mnt/nfs/mkv/{DISC_LABEL}/
├── title_00.mkv    # メインタイトル（映画本編など）
├── title_01.mkv    # ボーナス映像
├── title_02.mkv    # ...
└── ...
```

## 3. Phase 2: Transform & Load（MKV → HLS → MinIO + PostgreSQL）

### 3.1 概要

- NFS 上の MKV ファイルを入力として、ffmpeg で複数品質の HLS を生成。
- 生成した HLS ファイルを MinIO にアップロード。
- コンテンツメタデータを PostgreSQL に登録（→ Benthos で MongoDB, Elasticsearch に同期）。
- このフェーズは物理デバイス不要のため、任意のノードで実行可能。

### 3.2 品質バリアント

| バリアント | 解像度 | 映像ビットレート | 音声 |
| :--- | :--- | :--- | :--- |
| 1080p | 1920x1080 | 6 Mbps | AAC 192 kbps |
| 720p | 1280x720 | 3 Mbps | AAC 128 kbps |
| 480p | 854x480 | 1.5 Mbps | AAC 128 kbps |
| audio | - | - | AAC 192 kbps |
| master | - | マスタープレイリスト | - |

### 3.3 ffmpeg コマンド

```bash
ffmpeg -i "${INPUT_MKV}" \
  -filter_complex \
    "[0:v]split=3[v1][v2][v3]; \
     [v1]scale=1920:1080[v1out]; \
     [v2]scale=1280:720[v2out]; \
     [v3]scale=854:480[v3out]" \
  \
  -map "[v1out]" -c:v libx264 -preset fast -b:v 6000k -maxrate 6500k -bufsize 12000k \
  -map 0:a:0   -c:a aac -b:a 192k \
  -f hls -hls_time 6 -hls_list_size 0 \
  -hls_segment_filename "${OUTPUT_DIR}/1080p_%04d.ts" "${OUTPUT_DIR}/1080p.m3u8" \
  \
  -map "[v2out]" -c:v libx264 -preset fast -b:v 3000k -maxrate 3500k -bufsize 6000k \
  -map 0:a:0   -c:a aac -b:a 128k \
  -f hls -hls_time 6 -hls_list_size 0 \
  -hls_segment_filename "${OUTPUT_DIR}/720p_%04d.ts" "${OUTPUT_DIR}/720p.m3u8" \
  \
  -map "[v3out]" -c:v libx264 -preset fast -b:v 1500k -maxrate 2000k -bufsize 3000k \
  -map 0:a:0   -c:a aac -b:a 128k \
  -f hls -hls_time 6 -hls_list_size 0 \
  -hls_segment_filename "${OUTPUT_DIR}/480p_%04d.ts" "${OUTPUT_DIR}/480p.m3u8" \
  \
  -map 0:a:0 -vn -c:a aac -b:a 192k \
  -f hls -hls_time 6 -hls_list_size 0 \
  -hls_segment_filename "${OUTPUT_DIR}/audio_%04d.ts" "${OUTPUT_DIR}/audio.m3u8"
```

**GPU 対応:** NVIDIA GPU 搭載ノードでは `-c:v libx264` を `-c:v h264_nvenc -preset p4` に切り替え。

### 3.4 マスタープレイリスト生成

```bash
cat > "${OUTPUT_DIR}/master.m3u8" <<EOF
#EXTM3U
#EXT-X-VERSION:3

#EXT-X-STREAM-INF:BANDWIDTH=6192000,RESOLUTION=1920x1080,CODECS="avc1.640028,mp4a.40.2"
1080p.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=3128000,RESOLUTION=1280x720,CODECS="avc1.4d001f,mp4a.40.2"
720p.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=1628000,RESOLUTION=854x480,CODECS="avc1.42e01f,mp4a.40.2"
480p.m3u8

#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",NAME="Audio Only",DEFAULT=NO,URI="audio.m3u8"
EOF
```

### 3.5 MinIO アップロード

```bash
# HLS ファイルを MinIO にアップロード
mc cp --recursive "${OUTPUT_DIR}/" "minio/${BUCKET}/hls/${SHORT_ID}/"
```

### 3.6 PostgreSQL 登録

```sql
-- コンテンツ登録
INSERT INTO contents (short_id, disc_id, title, status)
VALUES ('${SHORT_ID}', '${DISC_ID}', '${TITLE}', 'processing');

-- バリアント登録
INSERT INTO video_variants (content_id, variant_type, hls_key, bandwidth, resolution)
VALUES
  ('${CONTENT_ID}', 'master', 'hls/${SHORT_ID}/master.m3u8', NULL, NULL),
  ('${CONTENT_ID}', '1080p',  'hls/${SHORT_ID}/1080p.m3u8',  6192000, '1920x1080'),
  ('${CONTENT_ID}', '720p',   'hls/${SHORT_ID}/720p.m3u8',   3128000, '1280x720'),
  ('${CONTENT_ID}', '480p',   'hls/${SHORT_ID}/480p.m3u8',   1628000, '854x480'),
  ('${CONTENT_ID}', 'audio',  'hls/${SHORT_ID}/audio.m3u8',  192000,  NULL);

-- ステータス更新
UPDATE contents SET status = 'ready', published_at = NOW() WHERE id = '${CONTENT_ID}';
```

登録後、Benthos CDC パイプラインが PostgreSQL の変更を検知し、MongoDB と Elasticsearch に自動同期する。

## 4. K8s Job 構成

### 4.1 Phase 1 Job（Extract）

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: bluray-extract-{disc-label}
spec:
  template:
    spec:
      nodeName: {bluray-node}    # Bluray ドライブ搭載ノード
      containers:
        - name: makemkv
          image: makemkv:latest
          volumeMounts:
            - name: nfs-mkv
              mountPath: /mnt/nfs/mkv
          securityContext:
            privileged: true     # /dev/sr0 アクセスのため
      volumes:
        - name: nfs-mkv
          nfs:
            server: {nfs-server}
            path: /mkv
      restartPolicy: Never
```

### 4.2 Phase 2 Job（Transform & Load）

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: transcode-{short-id}
spec:
  template:
    spec:
      containers:
        - name: ffmpeg
          image: ffmpeg-hls:latest
          env:
            - name: INPUT_MKV
              value: "/mnt/nfs/mkv/{disc-label}/{title}.mkv"
            - name: OUTPUT_DIR
              value: "/tmp/hls"
            - name: SHORT_ID
              value: "{short-id}"
          volumeMounts:
            - name: nfs-mkv
              mountPath: /mnt/nfs/mkv
              readOnly: true
      volumes:
        - name: nfs-mkv
          nfs:
            server: {nfs-server}
            path: /mkv
      restartPolicy: Never
```

## 5. オーケストレーション

初期実装は K8s Job を手動またはスクリプトで実行。  
並列処理の必要性が高まった場合は Argo Workflow への移行を検討する。

**Phase 1 → Phase 2 の連携:**
- Phase 1 Job 完了後にスクリプトが Phase 2 Job を生成（各 MKV タイトルごとに1 Job）。
- または Argo Workflow の DAG でフェーズを定義。

## 6. 未解決課題・将来対応

- **GPU トランスコード**: h264_nvenc 対応のための Node Selector / taint 設定。
- **進捗監視**: トランスコード進捗を PostgreSQL に記録し、管理 UI で確認できるようにする。
- **エラーハンドリング**: Job 失敗時の再実行戦略（MKV は保持しているので再実行可能）。
- **複数音声トラック**: MKV に複数音声トラックがある場合の扱い（字幕・多言語対応）。
