# Bluray ディスク取り込みパイプライン設計（v2）

## 1. 概要

物理 Bluray ディスクをドライブに挿入し、最終的に Web ブラウザで ABR ストリーミング再生できる状態にするための ETL パイプラインを定義する。

**v2 での主な変更点:**
- ABR（Adaptive Bitrate Streaming）: 1080p / 720p / 480p + 音声のみの複数品質 HLS を生成。
- MKV 保持: HLS 変換後も NFS 上の MKV ファイルは削除せず保持。
- ワークフロー分割: 物理デバイス依存の Phase 1 と、それ以降の Phase 2/3 を明確に分離。
- 1 ディスク複数コンテンツ対応: MakeMKV が生成する複数タイトルをそれぞれ独立したコンテンツとして登録。

---

## 2. パイプライン全体フロー

```
[Phase 1: Extract（物理デバイス依存）]
    Bluray ドライブ
        │ blkid でラベル検出
        ▼
    MakeMKV
        │ disc:1 all → NFS /mnt/nfs/mkv/{LABEL}/
        ▼
    NFS (mkv/) ← MKV を永続保持（再変換可能なように）

[Phase 2: Transform（NFS ファイル操作）]
    NFS (mkv/) 内の各 .mkv ファイル
        │ 未処理ファイルを検出
        ▼
    ffmpeg（ABR トランスコード）
        ├─ 1080p HLS（映像+音声）
        ├─ 720p  HLS（映像+音声）
        ├─ 480p  HLS（映像+音声）
        ├─ audio HLS（音声のみ）
        └─ master.m3u8（マスタープレイリスト）
        │
        ▼
    MinIO アップロード（hls/{short_id}/...）

[Phase 3: Load（カタログ登録）]
    MinIO アップロード完了後
        ├─ Thumbnail Analyzer → サムネイル生成 → MinIO
        └─ PostgreSQL 登録（discs / contents / video_variants / assets）
```

---

## 3. 各ステップの詳細設計

### 3.1 Phase 1: Bluray → MKV（Extract）

**ツール**: MakeMKV（Docker イメージ: `jlesage/makemkv`）

**処理内容:**
1. `blkid /dev/sr1` でディスクラベル（`LABEL`）を取得。
2. `LABEL` が空の場合は処理をスキップ（ディスクなし）。
3. 出力先 `/mnt/nfs/mkv/{LABEL}/` が既に存在する場合はスキップ（冪等性確保）。
4. `makemkvcon mkv disc:1 all /mnt/nfs/mkv/{LABEL}/` を実行。
5. 処理完了後にディスクをイジェクト（`eject /dev/sr1`）。
6. 後続の Phase 2/3 をトリガー（K8s Job 作成 または Argo Workflow 起動）。

**ハードウェア要件:**
- Bluray ドライブが接続されたノードが K8s クラスタに存在すること。
- ノードに `bluray-disk-device: "true"` ラベルを付与すること。
- コンテナは `privileged: true` が必要（デバイスアクセスのため）。

**出力例:**
```
/mnt/nfs/mkv/MY_MOVIE_DISC/
    title_00.mkv   ← 本編（最長タイトル）
    title_01.mkv   ← 特典映像
    title_02.mkv   ← 予告編
    ...
```

**注意:** コピーガード（AACS/BD+）が有効なディスクでは動作しない。個人バックアップ用途に限ること。MakeMKV ライセンスキーは `MAKEMKV_KEY` 環境変数で設定。

---

### 3.2 Phase 2: MKV → HLS ABR（Transform）

**ツール**: ffmpeg

**処理対象:** Phase 1 で生成された `/mnt/nfs/mkv/{LABEL}/` 内の各 `.mkv` ファイル。

**処理内容:**
1. 各 `.mkv` に対して ffmpeg で ABR HLS を生成。
2. 出力先: `/mnt/nfs/hls/{LABEL}/{title_name}/`（NFS 一時領域）。
3. 生成後、MinIO にアップロード（`hls/{short_id}/`）。

**生成品質一覧:**

| バリアント | 解像度 | 映像ビットレート | 音声ビットレート | コーデック |
| :--- | :--- | :--- | :--- | :--- |
| 1080p | 1920×1080 | 6,000 kbps | 192 kbps | H.264 + AAC |
| 720p | 1280×720 | 3,000 kbps | 128 kbps | H.264 + AAC |
| 480p | 854×480 | 1,500 kbps | 128 kbps | H.264 + AAC |
| audio | - | - | 192 kbps | AAC のみ |

**ffmpeg コマンド (CPU エンコード):**
```bash
ffmpeg -i input.mkv \
  -filter_complex \
    "[0:v]split=3[v1][v2][v3]; \
     [v1]scale=1920:1080[v1out]; \
     [v2]scale=1280:720[v2out]; \
     [v3]scale=854:480[v3out]" \
  \
  -map "[v1out]" -c:v libx264 -preset fast -b:v 6000k -maxrate 6500k -bufsize 12000k \
  -map 0:a:0 -c:a aac -b:a 192k \
  -f hls -hls_time 6 -hls_list_size 0 -hls_segment_filename "1080p_%04d.ts" 1080p.m3u8 \
  \
  -map "[v2out]" -c:v libx264 -preset fast -b:v 3000k -maxrate 3500k -bufsize 6000k \
  -map 0:a:0 -c:a aac -b:a 128k \
  -f hls -hls_time 6 -hls_list_size 0 -hls_segment_filename "720p_%04d.ts" 720p.m3u8 \
  \
  -map "[v3out]" -c:v libx264 -preset fast -b:v 1500k -maxrate 2000k -bufsize 3000k \
  -map 0:a:0 -c:a aac -b:a 128k \
  -f hls -hls_time 6 -hls_list_size 0 -hls_segment_filename "480p_%04d.ts" 480p.m3u8 \
  \
  -map 0:a:0 -vn -c:a aac -b:a 192k \
  -f hls -hls_time 6 -hls_list_size 0 -hls_segment_filename "audio_%04d.ts" audio.m3u8
```

**GPU 対応 (NVIDIA h264_nvenc):**
```bash
# -c:v libx264 を以下に置き換え
-c:v h264_nvenc -preset p4 -tune hq
```

**マスタープレイリスト生成 (スクリプトで自動生成):**
```m3u8
#EXTM3U
#EXT-X-VERSION:3

#EXT-X-STREAM-INF:BANDWIDTH=6192000,RESOLUTION=1920x1080,CODECS="avc1.640028,mp4a.40.2"
1080p.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=3128000,RESOLUTION=1280x720,CODECS="avc1.4d001f,mp4a.40.2"
720p.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=1628000,RESOLUTION=854x480,CODECS="avc1.42e01f,mp4a.40.2"
480p.m3u8

#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",NAME="Audio Only",DEFAULT=NO,URI="audio.m3u8"
```

---

### 3.3 Phase 3: MinIO + PostgreSQL（Load）

**処理内容:**
1. Phase 2 の出力ディレクトリ（NFS HLS）を MinIO にアップロード。
   - オブジェクトキー: `hls/{short_id}/{ファイル名}`
   - セグメント（`.ts`）は並列アップロード（goroutine）。
2. Thumbnail Analyzer サイドカーを呼び出し、MKV または HLS から最適フレームを抽出。
   - サムネイル画像を MinIO にアップロード: `thumbnails/{short_id}/thumb_{N}.jpg`
3. PostgreSQL に登録:
   - `discs` テーブル: ディスクエントリ（未登録の場合）。
   - `contents` テーブル: 各タイトルのエントリ（status: `ready`）。
   - `video_variants` テーブル: master / 1080p / 720p / 480p / audio の各エントリ。
   - `assets` テーブル: サムネイル。

**初期登録時のデフォルト値:**

| フィールド | デフォルト | 備考 |
| :--- | :--- | :--- |
| `title` | ファイル名（拡張子除く） | 管理画面で後から変更 |
| `description` | `""` | 管理画面で後から変更 |
| `content_type` | `video` | |
| `is_360` | `false` | |
| `status` | `ready` | |
| `published_at` | 登録日時 | |

---

## 4. オーケストレーション

### 4.1 ワークフロー分割の理由

| フェーズ | 物理デバイス依存 | 実行環境 |
| :--- | :--- | :--- |
| Phase 1 (Extract) | あり（Bluray ドライブ） | `bluray-disk-device=true` ノード固定 |
| Phase 2/3 (Transform+Load) | なし（NFS + MinIO） | 任意のノード（リソース次第） |

### 4.2 Argo Workflow vs K8s CronJob

| 観点 | Argo Workflow | K8s CronJob |
| :--- | :--- | :--- |
| DAG 依存関係 | ネイティブサポート | 複雑（複数 Job の連携が手間） |
| 並列処理 | 容易（parallelism） | 困難 |
| ログ/モニタリング | UI あり | kubectl logs |
| 運用コスト | Argo インストール必要 | K8s 標準 |
| **判定** | タイトル数が多い場合や並列変換が必要なら Argo を採用 | シンプルな逐次処理なら CronJob で十分 |

**推奨**: まず K8s CronJob (Phase 1) + K8s Job (Phase 2/3) で実装し、並列処理の必要性が出た時点で Argo に移行する。

### 4.3 処理フロー（K8s Job ベース）

```
K8s CronJob (*/5 * * * *)
    └─ Phase 1 Job (bluray-extract)
           条件: ディスクあり
           ノード: bluray-disk-device=true
           出力: /mnt/nfs/mkv/{LABEL}/
               │
               └─ 後続 Job をトリガー (K8s Job API 経由)

K8s Job (transform-load-{LABEL})
    ├─ Phase 2: ffmpeg（NFS → NFS hls/）
    └─ Phase 3: NFS Importer（NFS hls/ → MinIO → PostgreSQL）
```

---

## 5. ストレージ容量見積もり

| 項目 | 1 タイトルあたり | 備考 |
| :--- | :--- | :--- |
| MKV（NFS 保持） | 20〜50 GB | Bluray 本編の場合 |
| HLS 全品質合計 | 5〜15 GB | 1080p+720p+480p+audio |
| サムネイル | < 1 MB | |

---

## 6. 冪等性・エラー回復

- Phase 1: 出力ディレクトリが既に存在すればスキップ（再実行安全）。
- Phase 2: `video_variants` テーブルに `hls_key` が登録済みならスキップ。
- Phase 3: MinIO の `ObjectExists` チェック後にアップロード（重複防止）。
- エラー時: `contents.status` を `error` に設定し、ログに詳細を記録。再実行はステータスリセット後に可能。
