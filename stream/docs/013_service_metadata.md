# Metadata Service 設計詳細 (service_metadata.md)

## 1. サービスの目的と責務
動画コンテンツ、ジャンル、関連アセットの「正」となる情報(Master Data)を管理する。主にAdmin（管理者）向けのCRUD機能を提供し、データはPostgreSQLに保存する。更新が発生した場合は、同期的にCatalog Serviceへ反映(Sync Strategy A)する。

## 2. API エンドポイント定義

*全てのエンドポイントはAdmin権限（JWTに`role: admin`が含まれること）を要求する。*

### `POST /api/v1/admin/contents`
- **目的**: 新しいコンテンツのマスター作成（動画、画像群、360度動画など）
- **Payload**:
  ```json
  {
    "content_type": "video",
    "title": "Super Sci-Fi Video",
    "description": "An amazing journey...",
    "video_details": {
      "is_360": false,
      "duration_seconds": 5400,
      "director": "Jane Doe"
    }
  }
  ```
- **処理**: `contents` ベーステーブルにINSERT後、`content_type` に応じて `content_videos` 等の専用テーブルにINSERTする。
- **Response**: 採番された内部UUIDと外部公開用Short ID(`short_id`)を返却。

### `PUT /api/v1/admin/contents/:id`
- **目的**: 基本情報の更新。

### `POST /api/v1/admin/contents/:id/assets`
- **目的**: S3等のアップロード完了後、そのファイルパス(キー)を様々な目的(`asset_type`)でDBに登録・紐付ける。
- **Payload**:
  ```json
  {
    "assets": [
      { "asset_type": "poster", "s3_key": "assets/images/V7qL9x_poster.jpg" },
      { "asset_type": "hls_master", "s3_key": "assets/hls/V7qL9x/master.m3u8" }
    ]
  }
  ```
- **処理**: DB(assetsテーブル)更新後、Catalog Serviceの `POST /api/internal/catalog/sync/:id` をコールしてRead側へ同期(Sync A)する。

### `GET /api/v1/admin/genres` (Planned)
- **目的**: ジャンル一覧取得（将来的な機能）。

## 3. IDの取り扱い
- クライアントへのレスポンスには常に `short_id` を含める。
- S3へのアップロード時のファイル名・ディレクトリプレフィックスには `short_id` を用いる（UUIDは長すぎて扱いにくいため）。
