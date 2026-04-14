# ストリーミングサービス システムアーキテクチャ設計書

`101_requirements.md` の要件を満たすためのアーキテクチャを定義する。**シンプルさ** と **性能** を両立させる。

## 1. 設計方針

| 項目 | 方針 |
| :--- | :--- |
| 認証・認可 | **廃止**（個人利用のため不要） |
| 書き込み DB | PostgreSQL（Single Source of Truth） |
| 読み取り DB | MongoDB（Benthos により PostgreSQL から同期、高速読み取り最適化） |
| 全文検索 | Elasticsearch（PostgreSQL から同期） |
| HLS 品質 | ABR（1080p / 720p / 480p + audio のみ） |
| キャッシュ | API インメモリキャッシュ + Nginx HLS キャッシュ |

## 2. 全体アーキテクチャ

```
[ブラウザ]
    │
    ▼
[Nginx / Caddy (リバースプロキシ + HLS キャッシュ)]
    ├─ /api/*  → API Service (Go)
    ├─ /*      → Frontend (Next.js)
    └─ /hls/*  → MinIO (HLS セグメント直接配信)

[API Service (Go)]
    ├─ MongoDB     (コンテンツ一覧・詳細取得 — 読み取り最適化)
    ├─ Elasticsearch (検索クエリ)
    ├─ PostgreSQL  (書き込み・管理操作)
    └─ MinIO       (Presigned URL 生成 / サムネイルアップロード)

[Benthos (Redpanda Connect)]
    └─ PostgreSQL → MongoDB 同期パイプライン
       （INSERT/UPDATE/DELETE イベントをキャプチャし MongoDB へ反映）

[Batch: NFS Importer (Go)]
    ├─ NFS         (HLS ファイルスキャン)
    ├─ MinIO       (HLS セグメントアップロード)
    ├─ PostgreSQL  (コンテンツ登録)
    └─ Thumbnail Analyzer (Python サイドカー)

[Batch: ETL Pipeline (K8s Job / Argo Workflow)]
    ├─ Phase 1: Bluray Drive → MakeMKV → NFS (mkv/)
    └─ Phase 2: NFS (mkv/) → ffmpeg ABR → MinIO (hls/) + PostgreSQL
```

## 3. サービス設計

### 3.1 API Service (Go)

単一の Go サービスが全 API を提供する。

**読み取り操作（MongoDB 経由）:**

| メソッド | パス | 説明 |
| :--- | :--- | :--- |
| GET | `/v1/catalog` | コンテンツ一覧（フィルタ・ページング） |
| GET | `/v1/catalog/:short_id` | コンテンツ詳細 |
| GET | `/v1/catalog/:short_id/variants` | 動画バリアント一覧（ABR プレーヤー用） |
| GET | `/v1/search?q=...` | キーワード検索（Elasticsearch） |

**書き込み操作（PostgreSQL 経由）:**

| メソッド | パス | 説明 |
| :--- | :--- | :--- |
| POST | `/v1/admin/contents` | コンテンツ登録 |
| PUT | `/v1/admin/contents/:id` | コンテンツ更新 |
| DELETE | `/v1/admin/contents/:id` | コンテンツ削除 |
| GET | `/v1/admin/contents` | 管理用一覧（status 含む） |
| GET | `/v1/admin/discs` | ディスク一覧 |

**性能施策:**
- カタログ一覧は API インメモリキャッシュ（TTL 30 秒）。コンテンツ更新時に invalidate。
- MongoDB は非正規化ドキュメント構造で高速読み取りを実現（JOIN 不要）。
- `short_id` は Snowflake ID（分散環境での一意性・順序性確保）。

### 3.2 MongoDB ドキュメント設計

```json
// contents コレクション（非正規化、読み取り最適化）
{
  "_id": "01HXXX...",          // Snowflake ID (= short_id)
  "content_type": "video",
  "title": "タイトル名",
  "description": "説明",
  "duration_seconds": 7200,
  "is_360": false,
  "tags": ["action", "4k"],
  "status": "ready",
  "disc_label": "DISC_001",   // 非正規化（JOIN 不要）
  "variants": [               // 非正規化（JOIN 不要）
    {
      "variant_type": "master",
      "hls_key": "hls/01HXXX/master.m3u8"
    },
    {
      "variant_type": "1080p",
      "hls_key": "hls/01HXXX/1080p.m3u8",
      "bandwidth": 6192000,
      "resolution": "1920x1080"
    }
  ],
  "thumbnail_key": "thumbnails/01HXXX/thumb_01.jpg",
  "published_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

### 3.3 Benthos (Redpanda Connect) 同期パイプライン

PostgreSQL → MongoDB の変更データキャプチャ（CDC）。

```yaml
# benthos-pg-to-mongo.yaml（概略）
input:
  postgres_cdc:
    dsn: "${POSTGRES_DSN}"
    slot_name: "benthos_replication"
    tables:
      - contents
      - video_variants
      - assets
      - discs

pipeline:
  processors:
    - mapping: |
        # PostgreSQL の正規化データを MongoDB の非正規化ドキュメントに変換
        root = this.apply("flatten_content")

output:
  mongodb:
    url: "${MONGO_URL}"
    database: "stream"
    collection: "contents"
    operation: "upsert"
    filter_map: 'root = {"_id": this.short_id}'
    document_map: "root = this"
```

### 3.4 Elasticsearch インデックス設計

```json
// index: stream_contents
{
  "mappings": {
    "properties": {
      "short_id":    { "type": "keyword" },
      "title":       { "type": "text", "analyzer": "kuromoji" },
      "description": { "type": "text", "analyzer": "kuromoji" },
      "tags":        { "type": "keyword" },
      "content_type":{ "type": "keyword" },
      "status":      { "type": "keyword" },
      "updated_at":  { "type": "date" }
    }
  }
}
```

同期方法: Benthos パイプラインの出力に Elasticsearch への書き込みを追加（fan-out）。

### 3.5 Frontend (Next.js)

- ログイン/認証 UI を全廃。
- ABR プレーヤー: `hls.js` を使用。品質選択 UI を提供。
- 360度動画: `three.js` または `A-Frame` ベースのビューア。

### 3.6 Thumbnail Analyzer (Python バッチ)

MKV または HLS から複数フレームをサンプリングし、輝度スコアで最適なサムネイルを選択。

## 4. データベース設計（PostgreSQL）

```sql
-- =====================
-- Bluray ディスク管理
-- =====================
CREATE TABLE discs (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    label      VARCHAR(255) NOT NULL UNIQUE,  -- ディスクラベル (blkid から取得)
    disc_name  VARCHAR(255),                  -- 管理画面で設定する名前
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =====================
-- コンテンツ（各タイトル）
-- =====================
CREATE TABLE contents (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    short_id         VARCHAR(50)  UNIQUE NOT NULL,  -- Snowflake ID（外部公開用）
    content_type     VARCHAR(20)  NOT NULL DEFAULT 'video',  -- 'video', 'image_gallery', 'vr360'
    disc_id          UUID         REFERENCES discs(id),      -- NULL = 手動登録
    title            VARCHAR(255) NOT NULL,
    description      TEXT         DEFAULT '',
    duration_seconds INTEGER,
    is_360           BOOLEAN      DEFAULT FALSE,
    tags             TEXT[]       DEFAULT '{}',
    status           VARCHAR(20)  NOT NULL DEFAULT 'pending',
    -- status: pending / processing / ready / error
    published_at     TIMESTAMP WITH TIME ZONE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_contents_short_id   ON contents(short_id);
CREATE INDEX idx_contents_status     ON contents(status);
CREATE INDEX idx_contents_type       ON contents(content_type);
CREATE INDEX idx_contents_updated_at ON contents(updated_at);

-- =====================
-- 動画バリアント（ABR 各品質）
-- =====================
CREATE TABLE video_variants (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    content_id   UUID         NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    variant_type VARCHAR(20)  NOT NULL,
    -- 'master'(マスタープレイリスト), '1080p', '720p', '480p', 'audio'
    hls_key      TEXT         NOT NULL,  -- MinIO オブジェクトキー（.m3u8）
    bandwidth    INTEGER,                -- bps（マスタープレイリスト用）
    resolution   VARCHAR(20),           -- '1920x1080' など（audio は NULL）
    codecs       VARCHAR(100),          -- 'avc1.640028,mp4a.40.2' など
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_video_variants_content_id ON video_variants(content_id);

-- =====================
-- アセット（サムネイル・ポスター等）
-- =====================
CREATE TABLE assets (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    content_id UUID        NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    asset_role VARCHAR(50) NOT NULL,  -- 'thumbnail', 'poster'
    s3_key     TEXT        NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_assets_content_id ON assets(content_id);
```

## 5. ストレージ設計

### 5.1 NFS（中間ストレージ）

```
/mnt/nfs/
├── mkv/
│   └── {DISC_LABEL}/         # MakeMKV 出力（永続保持）
│       ├── title_00.mkv
│       ├── title_01.mkv
│       └── ...
└── hls/                       # ffmpeg 出力（MinIO アップロード後は削除可）
    └── {DISC_LABEL}/
        └── {title_name}/
            ├── master.m3u8
            ├── 1080p.m3u8
            ├── 1080p_000.ts
            └── ...
```

### 5.2 MinIO（配信ストレージ）

```
{bucket}/
├── hls/
│   └── {short_id}/
│       ├── master.m3u8
│       ├── 1080p.m3u8
│       ├── 1080p_000.ts
│       ├── 720p.m3u8
│       ├── 720p_000.ts
│       ├── 480p.m3u8
│       ├── 480p_000.ts
│       ├── audio.m3u8
│       └── audio_000.ts
└── thumbnails/
    └── {short_id}/
        ├── thumb_01.jpg
        └── thumb_02.jpg
```

## 6. HLS / ABR 設計

### 6.1 ffmpeg トランスコードコマンド

```bash
ffmpeg -i input.mkv \
  -filter_complex \
    "[0:v]split=3[v1][v2][v3]; \
     [v1]scale=1920:1080[v1out]; \
     [v2]scale=1280:720[v2out]; \
     [v3]scale=854:480[v3out]" \
  \
  -map "[v1out]" -c:v libx264 -preset fast -b:v 6000k -maxrate 6500k -bufsize 12000k \
  -map 0:a:0   -c:a aac -b:a 192k \
  -f hls -hls_time 6 -hls_list_size 0 \
  -hls_segment_filename "1080p_%04d.ts" 1080p.m3u8 \
  \
  -map "[v2out]" -c:v libx264 -preset fast -b:v 3000k -maxrate 3500k -bufsize 6000k \
  -map 0:a:0   -c:a aac -b:a 128k \
  -f hls -hls_time 6 -hls_list_size 0 \
  -hls_segment_filename "720p_%04d.ts" 720p.m3u8 \
  \
  -map "[v3out]" -c:v libx264 -preset fast -b:v 1500k -maxrate 2000k -bufsize 3000k \
  -map 0:a:0   -c:a aac -b:a 128k \
  -f hls -hls_time 6 -hls_list_size 0 \
  -hls_segment_filename "480p_%04d.ts" 480p.m3u8 \
  \
  -map 0:a:0 -vn -c:a aac -b:a 192k \
  -f hls -hls_time 6 -hls_list_size 0 \
  -hls_segment_filename "audio_%04d.ts" audio.m3u8
```

**GPU 対応 (h264_nvenc):** NVIDIA GPU が利用可能な場合は `-c:v libx264` を `-c:v h264_nvenc -preset p4` に切り替える。

### 6.2 マスタープレイリスト (.m3u8)

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

## 7. キャッシュ戦略

| 対象 | キャッシュ層 | TTL |
| :--- | :--- | :--- |
| HLS セグメント (`.ts`) | Nginx | 1年（不変） |
| HLS マニフェスト (`.m3u8`) | Nginx | 30秒 |
| API カタログ一覧 | API インメモリ（Go sync.Map） | 30秒 |
| サムネイル・ポスター | Nginx | 1週間 |
| MongoDB ドキュメント | MongoDB（インメモリキャッシュ） | 常時 |

## 8. 実装フェーズ計画

### Phase 0（現在）: クリーンアップ + ドキュメント整備
- [x] 旧コードの全削除
- [x] 新要件・アーキテクチャ設計書の作成（MongoDB + Benthos + Elasticsearch 維持版）

### Phase 1: インフラ・DB 基盤
- PostgreSQL, MongoDB, Elasticsearch, MinIO の K8s マニフェスト作成
- Benthos CDC パイプライン設定
- PostgreSQL スキーマ（マイグレーション）実装

### Phase 2: API Service 実装（Go）
- CRUD エンドポイント実装（PostgreSQL 書き込み）
- 読み取りエンドポイント実装（MongoDB 経由）
- 検索エンドポイント実装（Elasticsearch）
- インメモリキャッシュ実装

### Phase 3: フロントエンド実装（Next.js）
- コンテンツ一覧・詳細画面
- `hls.js` ABR プレーヤー
- 管理画面（CMS）

### Phase 4: Bluray ETL パイプライン実装
- MakeMKV K8s Job（Phase 1: Extract）
- ffmpeg ABR トランスコード K8s Job（Phase 2: Transform）
- NFS Importer（Phase 3: Load）
- Thumbnail Analyzer

### Phase 5: 最適化・運用整備
- Nginx HLS キャッシュチューニング
- GPU トランスコード対応（h264_nvenc）
- 監視・ログ整備
