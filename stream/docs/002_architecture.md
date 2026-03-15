# マルチコンテンツ・ストリーミングサービス システムアーキテクチャ設計書 (Architecture Design)

本ドキュメントでは、`requirements.md` で定義された要件を満たすための具体的なシステム構成、データベース設計、およびインフラ構成を定義する。

## 1. 全体アーキテクチャ (System Landscape)

システム全体の基本方針は**「マイクロサービス化」**と**「API Gatewayパターン」**の採用。
現在は開発環境において、UIを `localhost:3000`、API Gateway(Nginx)を `localhost:8000` で提供するハイブリッド構成をとっている。

```mermaid
graph TD
    Client["Client Browser (localhost:3000)"]
    Gateway["Nginx API Gateway (localhost:8000)"]
    Frontend["Frontend Service (Next.js)"]
    Catalog["Catalog Service"]
    Metadata["Metadata Service"]
    Auth["Auth Service"]
    
    DB_PG[("PostgreSQL")]
    DB_Mongo[("MongoDB")]
    DB_Redis[("Redis")]

    Client -- UI Request --> Frontend
    Client -- Client-side API Request --> Gateway
    Frontend -- Server-side API Request --> Gateway
    
    Gateway -- /api/catalog/* --> Catalog
    Gateway -- /api/metadata/* --> Metadata
    Gateway -- /api/auth/* --> Auth
    
    Catalog --> DB_Mongo
    Catalog --> DB_Redis
    Metadata --> DB_PG
    Auth --> DB_PG

    subgraph Batch Layer
        NFS[("NFS Server (Video Source)")]
        Importer["Batch NFS Importer (Go)"]
        Analyzer["Thumbnail Analyzer (Python/Sidecar)"]
        SyncBatch["Sync Batch (Go)"]
        
        NFS -- Scan/Read --> Importer
        Importer -- Analyze Request (HTTP) --> Analyzer
        Importer -- S3 Upload --> MinIO[("MinIO (S3)")]
        Importer -- Write --> DB_PG
        
        SyncBatch -- Read --> DB_PG
        SyncBatch -- Sync --> DB_Mongo
        SyncBatch -- Sync --> DB_Redis
    end
```

### Data Flow (Development)
1.  **UI表示**: ブラウザから `localhost:3000` にアクセス。
2.  **Server-side Fetch**: Next.js サーバーが内部ネットワーク経由 (`http://nginx:80`) でAPI Gatewayを叩き、カタログ情報を取得。
3.  **Client-side Fetch**: ブラウザ（hls.js や 管理画面モーダル）が `localhost:8000` を経由して各APIや動画アセットを取得。
4.  **CORS**: Nginx Gatewayが `localhost:3000` からのリクエストを許可するヘッダーを付与。

## 2. サービス設計詳細

### 2.1 Auth Service (認証・認可)
- **役割**: エンドユーザーの登録、ログインセッション管理（JWT発行・検証）、およびAdminユーザーの権限管理を担う。
- **データストア**: PostgreSQL (ユーザーテーブル、セッショントークンテーブル等)
- **設計方針**: パスワードのハッシュ化(bcrypt)、JWTによるステートレスな認証機構。将来的にはOAuth等の外部認証プロバイダ追加も視野に入れる。

### 2.2 Metadata Service (メタデータ管理API)
- **役割**: 映像作品(Title)、エピソード、ジャンル、タグなどのマスターデータ(CRUD)と、アセットパス(S3バケットキー)を管理する。
- **データストア**: PostgreSQL (リレーショナル・トランザクション処理)
  - **Short ID (表示用ID)**: Snowflake IDを用いて生成され、クライアントへのレスポンスやURL(例: `/title/7214567890`)で使用される。分散環境での一意性と順序性を両立する。
- **同期処理 (Sync Strategy A - 同期呼出し)**:
  - Adminからのメタデータ更新（INSERT/UPDATE）処理完了時、Metadata ServiceはCatalog Serviceの「データ同期用API」を内部呼び出し(HTTP/gRPC)し、即座に更新を反映させる。
  - シンプルさを重視し、まずは非同期MQ(RabbitMQ等)は導入せず、同期によるAPI連携で開始する。

### 2.3 Catalog & Search Service (カタログ・検索API)
- **役割**: エンドユーザー（フロントエンド）向けの参照特化型API。画面描画に最適化されたペイロードを最速で返却する。
- **データストア**: MongoDB および Redis
- **設計方針**:
  - `MongoDB` へのRead: 各作品情報(Titles, Genres, Assetsパス等)を非正規化した一つのドキュメントとして保持し、結合クエリを排除。
  - `Redis` キャッシュ: トップページ（カルーセル）、人気ランキング、ジャンル別一覧など、全ユーザー共通のデータはRedisからミリ秒以下で返却。Metadata Serviceからの更新イベント(Sync A)を受けた際にキャッシュをパージ・再生成する。

### 2.4 インフラ・配信層 (CDN & Object Storage)
- **コンテンツ・画像管理**: アプリケーションサーバーやローカルディスクは利用せず、**MinIO (S3互換・セルフホスト可能)** を利用する。
- **高速配信**: 画像とHLSセグメントファイル(`.ts`, `.m3u8`)はCDNエッジでキャッシュ。アプリケーションバックエンドにはアクセスさせず、シーク・ローディング時のレイテンシを極小化する。

### 2.5 Batch Services & Sidecars
- **Batch NFS Importer (Go)**: NFS上のHLSファイルをスキャンし、MinIOへのアップロードとメタデータ登録を行う。
- **Thumbnail Analyzer (Python/FastAPI)**: Importerから呼び出されるサイドカーコンテナ。動画の輝度・音量などの重い解析ロジックを担当。
- **Sync Batch (Go)**: PostgreSQLのマスターデータ（Write系）を MongoDB/Redis（Read系）へ同期・パージするバッチプロセス。

- **設計方針**: 解析などのリソース消費が激しい処理や、Pythonのエコシステム（AI/画像処理）を活用したい処理をサイドカーとして分離することで、メインのバッチ処理の安定性と拡張性を確保する。また、書き込み系（RDB）と参照系（NoSQL/Cache）を分離するCQRS構成において、同期漏れを補完するための結果整合性をバッチ処理によって担保する。

## 3. データベーススキーマ設計方針 (PostgreSQL)

以下に代表的なテーブルの概要（UUIDとShortIDの分離を考慮した形）を示す。

**`users` (認証用: Auth Service管轄)**
- `id` (UUID, PK)
- `email`, `password_hash`, etc.

**`contents` (全コンテンツ共通のベーステーブル: Metadata Service管轄)**
全てのドキュメントに共通する「ID」「タイトル」「公開状態」などの基本情報を管理する。
- `id` (UUID, PK) - 内部リレーション用
- `short_id` (String, Unique, Indexed) - **【外部公開用】**(Snowflake ID)
- `content_type` (Enum: `video`, `image_gallery`, `ebook`...) - 子テーブルの種別判定用
- `title` (String) - タイトル名
- `description` (Text) - あらすじや説明
- `published_at` (Timestamp)
- `created_at`, `updated_at`

*(※ パフォーマンスと型安全を確保するため、コンテンツ種別ごとに専用の子テーブルを切る「Class Table Inheritance」パターンを採用する)*

**`content_videos` (動画固有データ: Metadata Service管轄)**
- `content_id` (UUID, PK, FK to `contents.id`)
- `is_360` (Boolean)
- `duration_seconds` (Int)
- `director` (String)

*(※ 今後、ネットワーク速度に応じた動的解像度対応(ABR)の導入を検討。その際はマスタープレイリスト等を用いた構成となる。)*

**`assets` (実体ファイル管理テーブル: Metadata Service管轄)**
1つのコンテンツに複数のアセット(動画マニフェスト、ポスター、PDF等)が紐づく。
- `id` (UUID, PK)
- `content_id` (UUID, FK to `contents.id`)
- `asset_role` (VARCHAR)
- `s3_key` (String) - MinIOオブジェクトキー

### 3.2 データベース間のスキーマ整合性 (PostgreSQL -> MongoDB)

- **MongoDBのスキーマ設計方針**:
  - PostgreSQLのような正規化（ベーステーブル＋専用テーブル＋アセットテーブル）を**そのままMongoDBに持ち込まない**。
  - Mongoの最大の強みである「1回のREADで画面描画に必要な全データを取得する」形（非正規化）にする。
  - したがって、Mongo側の `ContentCatalog` コレクションは「PostgreSQLの `contents` + `content_videos` + `assets` をあらかじめJOINして作られた単一の大きなJSONドキュメント」となる。

*(Catalog ServiceはMongoの1つのドキュメントを読むだけで、UIが必要とする「タイトル」「動画再生時間」「HLSパス」「ポスター画像パス」のすべてを取得できる)*

---

## 4. データ同期と耐障害性 (Eventual Consistency & Resiliency)

CQRSアーキテクチャにおける PostgreSQL(Write) と MongoDB(Read) のデータ整合性を担保するため、以下の二段構えとする。

1. **Online Sync (オンライン同期)**
   - APIリクエストの延長線上で同期。Adminがデータを保存した直後、Metadata ServiceからCatalog Serviceの内部APIを叩き、MongoDBの対象ドキュメントをUpsertする。
   - メリット: 反映が早くシンプル。
   - デメリット: ネットワークエラーやCatalog Service側の一時的なダウン時に更新が漏れるリスクがある（不整合の発生）。

2. **Batch Sync / Reconciliation (バッチ同期による結果整合性の担保)**
   - 定期実行のCron Job / Workerプロセスが、PostgreSQL側の `updated_at` が直近N分以内のレコードをポーリングし、MongoDB側のデータと比較。差分や漏れがあれば非同期に修正同期(Reconciliation)を行う。
   - これにより、仮にオンライン同期が失敗しても「遅くとも次のバッチ実行時にはデータが修正される」という**結果整合性 (Eventual Consistency)** を強固にする。

---

## 5. 今後検討すべきアーキテクチャ (Future Considerations)

初期フェーズから作り込むとオーバースペックになるため一旦見送るが、中長期的に必要となるメジャーな技術要素。

- **全文検索エンジン (Elasticsearch 等)**
  - 現在はMongoDBのフルテキスト検索機能を用いる想定だが、データ量増加時や「表記ゆれ」「サジェスト」機能が求められた場合は専用のSearch Engineへと同期する戦略が必要。
- **コンテンツ種別の動的拡張**
  - 現在の Video, Image, 360VR 以外にも、ドキュメント(PDF)や音声(Audio)など、配信対象を広げられる設計を維持する。
- **メッセージキュー (Event Driven Architecture)**
  - 上記の「PostgreSQL -> MongoDB -> Elasticsearch」という段階的な同期を同期APIやBatchに頼るのが苦しくなったフェーズでは、RabbitMQやKafkaといったメッセージブローカーを導入。デッドレターキュー(DLQ)を用いた確実なイベント配信基盤(Event Sourcing/CQRS)へと進化させる。
- **論理削除 (Soft Delete)**
  - コンテンツを誤って削除した場合の復旧手段。RDB上で `deleted_at` カラムを持つ。この際、MinIO上の数十GBのHLSファイル等の物理削除のタイミングをどう設計するかが論点となる。
