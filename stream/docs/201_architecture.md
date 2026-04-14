# ストリーミングサービス システムアーキテクチャ設計書（v2 - 全面改訂）

`101_requirements.md` (v2) の要件を満たすための新アーキテクチャを定義する。旧設計から大幅に刷新し、**シンプルさ** と **性能** を両立させる。

## 1. 設計方針の変更点（旧 v1 との差分）

| 項目 | v1（旧） | v2（新） |
| :--- | :--- | :--- |
| 認証・認可 | JWT + RBAC（Auth Service） | **廃止**（個人利用） |
| サービス構成 | Auth / Metadata / Catalog（3サービス） | **単一 API Service** |
| 検索基盤 | Elasticsearch | **廃止**（PostgreSQL FTS で代替） |
| 読み取り最適化 | MongoDB（Benthos 同期） | **廃止**（PostgreSQL + API キャッシュ） |
| HLS 品質 | 単一品質 | **ABR（1080p/720p/480p + audio）** |
| コンテンツモデル | contents / content_videos / assets | **discs / contents / video_variants / assets** |
| 削除対象コード | - | users / groups / roles / permissions 全テーブル、auth middleware、JWT |

## 2. 全体アーキテクチャ

```
[ブラウザ]
    │
    ▼
[Nginx / Caddy (リバースプロキシ + キャッシュ)]
    ├─ /api/*  → API Service (Go)
    ├─ /*      → Frontend (Next.js)
    └─ /hls/*  → MinIO (HLS セグメント直接配信 / キャッシュ)

[API Service (Go)]
    ├─ PostgreSQL (メタデータ読み書き)
    └─ MinIO (Presigned URL 生成 / サムネイルアップロード)

[Batch: NFS Importer (Go)]
    ├─ NFS (HLS ファイルスキャン)
    ├─ MinIO (HLS セグメントアップロード)
    ├─ PostgreSQL (コンテンツ登録)
    └─ Thumbnail Analyzer (Python サイドカー)

[Batch: ETL Pipeline]
    ├─ Phase 1: Bluray Drive → MakeMKV → NFS (mkv/)
    └─ Phase 2: NFS (mkv/) → ffmpeg → MinIO (hls/) + PostgreSQL
```

**削除するコンポーネント:**
- Auth Service
- Catalog Service（API Service に統合）
- Metadata Service（API Service に統合）
- Dev Proxy（認証プロキシ機能）
- Benthos (Redpanda Connect)
- MongoDB
- Elasticsearch

## 3. サービス設計

### 3.1 API Service (Go)

単一の Go サービスが全 API を提供する。

**エンドポイント:**

| メソッド | パス | 説明 |
| :--- | :--- | :--- |
| GET | `/v1/catalog` | コンテンツ一覧（フィルタ・ページング対応） |
| GET | `/v1/catalog/:short_id` | コンテンツ詳細 |
| GET | `/v1/catalog/:short_id/variants` | 動画バリアント一覧（ABR プレーヤー用） |
| GET | `/v1/search?q=...` | キーワード検索（PostgreSQL FTS） |
| POST | `/v1/admin/contents` | コンテンツ登録 |
| PUT | `/v1/admin/contents/:id` | コンテンツ更新 |
| DELETE | `/v1/admin/contents/:id` | コンテンツ削除 |
| GET | `/v1/admin/contents` | 管理用一覧（status 含む） |
| GET | `/v1/admin/discs` | ディスク一覧 |

**性能施策:**
- カタログ一覧はインメモリキャッシュ（TTL 30 秒）。コンテンツ更新時に invalidate。
- PostgreSQL のインデックス最適化（`short_id`, `status`, `content_type`, `updated_at`）。
- `short_id` は Snowflake ID（分散環境での一意性・順序性確保）。

### 3.2 Frontend (Next.js)

- ログイン/認証 UI を全廃。
- ABR プレーヤー: `hls.js` を使用。品質選択 UI を提供。
- 360度動画: `three.js` または `A-Frame` ベースのビューア。

### 3.3 NFS Importer (Go バッチ)

`402_batch_nfs_importer.md` を更新して、新データモデルに対応させる。

- NFS HLS ディレクトリをスキャンし、未登録のコンテンツを検出。
- `video_variants` テーブルに各品質バリアントを登録。
- Thumbnail Analyzer サイドカーを呼び出してサムネイルを生成。

### 3.4 Thumbnail Analyzer (Python サイドカー)

変更なし。MKV または HLS から複数フレームをサンプリングし、輝度スコアで最適なサムネイルを選択。

## 4. データベース設計（PostgreSQL）

```sql
-- =====================
-- Bluray ディスク管理
-- =====================
CREATE TABLE discs (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    label      VARCHAR(255) NOT NULL,        -- ディスクラベル (blkid から取得)
    disc_name  VARCHAR(255),                 -- 人が付けた名前（管理画面で変更可）
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
-- 全文検索インデックス（PostgreSQL FTS）
CREATE INDEX idx_contents_fts ON contents
    USING gin(to_tsvector('japanese', title || ' ' || coalesce(description, '')));

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

**削除するテーブル:** `users`, `groups`, `user_groups`, `roles`, `group_roles`, `user_roles`, `permissions`, `role_permissions`, `content_group_permissions`, `content_videos`

## 5. ストレージ設計

### 5.1 NFS（中間ストレージ）

```
/mnt/nfs/
├── mkv/
│   └── {DISC_LABEL}/         # MakeMKV 出力 (永続保持)
│       ├── title_00.mkv
│       ├── title_01.mkv
│       └── ...
└── hls/                       # ffmpeg 出力（MinIO アップロード後も一時的に保持）
    └── {DISC_LABEL}/
        └── {title_name}/
            ├── master.m3u8
            ├── 1080p.m3u8
            ├── 1080p_000.ts
            ├── ...
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
  -map 0:a:0 -c:a aac -b:a 192k \
  -f hls -hls_time 6 -hls_list_size 0 \
  -hls_segment_filename "1080p_%04d.ts" 1080p.m3u8 \
  \
  -map "[v2out]" -c:v libx264 -preset fast -b:v 3000k -maxrate 3500k -bufsize 6000k \
  -map 0:a:0 -c:a aac -b:a 128k \
  -f hls -hls_time 6 -hls_list_size 0 \
  -hls_segment_filename "720p_%04d.ts" 720p.m3u8 \
  \
  -map "[v3out]" -c:v libx264 -preset fast -b:v 1500k -maxrate 2000k -bufsize 3000k \
  -map 0:a:0 -c:a aac -b:a 128k \
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
| HLS セグメント (`.ts`) | Nginx / Cloudflare | 1年（不変） |
| HLS マニフェスト (`.m3u8`) | Nginx | 30秒（更新される可能性あり） |
| API カタログ一覧 | API インメモリ（Go sync.Map） | 30秒 |
| サムネイル・ポスター | Nginx / Cloudflare | 1週間 |

## 8. 実装フェーズ計画

### Phase 0（現在）: アーキテクチャ確定・ドキュメント整備
- [x] 新要件・アーキテクチャ設計書の作成
- [ ] OWNER のレビューと承認

### Phase 1: クリーンアップ（旧コードの削除）
- Auth Service の削除
- Auth ミドルウェア・JWT 関連コードの削除
- Benthos 設定の削除
- `users`, `groups`, `roles` 等テーブルの削除
- テスト（e2e/api）の認証関連コードの削除

### Phase 2: 新データモデル実装
- 新 DB スキーマ (`discs`, `contents`, `video_variants`, `assets`) の実装
- NFS Importer の新モデル対応

### Phase 3: API Service リファクタ
- Auth / Metadata / Catalog の統合
- 新エンドポイント実装
- PostgreSQL FTS 検索実装
- インメモリキャッシュ実装

### Phase 4: Bluray ETL パイプライン実装
- ABR ffmpeg トランスコードスクリプト
- NFS Importer の `video_variants` 対応
- K8s Job / Argo Workflow 設計・実装

### Phase 5: フロントエンドリファクタ
- 認証 UI の削除
- `hls.js` ABR プレーヤーへの置き換え
- 品質選択 UI の実装
