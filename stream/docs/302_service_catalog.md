# Catalog & Search Service 設計詳細

## 1. サービスの目的と責務
エンドユーザー（クライアントUI）向けに最適化されたコンテンツカタログの提供と、高速な検索機能を提供する。
Metadata Serviceで管理されているリレーショナルなデータを非正規化してMongoDBへ保持する。

## 2. API エンドポイント定義

*主なエンドポイントは全ユーザー（未ログイン含む）からアクセス可能、あるいはJWTに基づく簡易な認可チェックを通すのみとする。*

### `GET /api/v1/catalog/home`
- **目的**: トップページ描画に必要なデータセット（カルーセル画像、人気ジャンル別リスト、最新追加コンテンツ等）を一括取得する。
- **Response**: **MongoDB**から取得したJSONツリーをそのまま返却。（BFF的な役割）
- **パフォーマンス**: 10ms 〜 50ms 目安。

### `GET /api/v1/catalog/contents/:short_id`
- **目的**: コンテンツ詳細画面（メタデータ、あらすじ、関連アセットの一覧）を描画するための取得。
- **Response**: **MongoDB** (`ContentCatalog` コレクション)から取得したドキュメント。
- **Payload**:
  ```json
  {
    "short_id": "V7qL9x",
    "type": "video",
    "title": "Super Sci-Fi Video",
    "description": "An amazing journey...",
    "metadata": {
      "release_year": 2026,
      "director": "Jane Doe"
    },
    "assets": {
      "poster_url": "https://cdn.example.com/assets/images/V7qL9x_poster.jpg",
      "hls_url": "https://cdn.example.com/assets/hls/V7qL9x/master.m3u8"
    }
  }
  ```
*(※ `hls_url` 等の配信ドメインはCDNのURLをプレフィックスとして結合する)*

### `GET /api/v1/catalog/search?q=keyword`
- **目的**: 全文検索。
- **処理**: Elasticsearch と MongoDB を組み合わせた検索エンジンを使用する。詳細は [304_service_search_design.md](304_service_search_design.md) を参照。

---

## 3. 内部向けAPI (Metadataからの同期受信用)

### `POST /api/internal/catalog/sync/:short_id`
- **目的**: Metadata Serviceがデータ更新を行った直後にコールされる。
- **役割**:
  1. 対象データの最新版をMetadata Serviceから取得(DB直接参照、またはAPI経由)し、MongoDBのドキュメントを再構築・Upsert(上書き保存)する。
  2. (必要であれば検索インデックスの再生成などをキックする)
