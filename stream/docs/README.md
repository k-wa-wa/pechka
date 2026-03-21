# Documentation Rule

このディレクトリ内では、以下の連番ルールに従ってドキュメントを管理します。

## 命名規則

ファイル名は `[カテゴリ番号][連番]_[内容].md` の形式とします。

### カテゴリ番号 (100の位)

| 番号 | カテゴリ | 内容 |
| :--- | :--- | :--- |
| **100** | **要件 (Requirements)** | システム全体の要件定義、機能・非機能要件 |
| **200** | **アーキテクチャ (Architecture)** | 全体設計、システム構成、データフロー、全体的な設計方針 |
| **300** | **サービス設計 (Service Design)** | 各マイクロサービスの詳細設計（API、内部ロジック、データモデル） |
| **400** | **機能・詳細設計 (Component/Feature)** | フロントエンド、バッチ、特定のサブシステムや機能の詳細仕様 |
| **500** | **インターフェース (Interface)** | APIスペック (OpenAPI/Swagger)、通信プロトコル仕様 |
| **800** | **運用 (Operations)** | 運用手順、監視、トラブルシューティング、リリースフロー |
| **900** | **開発 (Development)** | 開発環境の構築、コーディング規約、ブランチ運用ルール |

### 連番 (10, 1の位)

- ドキュメントの作成順、または論理的な優先順位に従って付与します。

## ドキュメント一覧

- **101**: [Requirements (要件定義)](101_requirements.md)
- **201**: [Architecture (アーキテクチャ設計)](201_architecture.md)
- **202**: [Authorization (認可設計)](202_authorization.md)
- **301**: [Auth Service (認証サービス)](301_service_auth.md)
- **302**: [Catalog Service (カタログサービス)](302_service_catalog.md)
- **303**: [Metadata Service (メタデータサービス)](303_service_metadata.md)
- **304**: [Search Service (検索サービス設計)](304_service_search_design.md)
- **305**: [Dev Proxy Service (開発用プロキシ & 認証モック)](305_service_dev_proxy.md)
- **401**: [Frontend & UI (フロントエンド・UI仕様)](401_frontend_ui_spec.md)
- **402**: [Batch NFS Importer (バッチインポーター)](402_batch_nfs_importer.md)
- **403**: [Tagging Design (タグ機能設計)](403_tagging_design.md)
- **501**: [API Specification (API仕様)](501_api_spec.yaml)
- **801**: [Operations (運用マニュアル)](801_operations.md)
- **901**: [Development (開発ガイドライン)](901_development.md)
