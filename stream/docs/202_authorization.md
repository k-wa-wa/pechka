# 202 Authorization (認可設計)

本ドキュメントでは、システム全体の権限制御（誰がどの操作を行えるか、どのリソースを閲覧できるか）の設計方針について定義します。

## 1. 認可モデル (RBAC + ACL)

柔軟な権限管理とパフォーマンスを両立するため、**ロールベース (RBAC)** と **リソース単位 (ACL)** を組み合わせたハイブリッド方式を採用します。

| 方式 | 適用対象 | 判定タイミング |
| :--- | :--- | :--- |
| **RBAC** | システム全体の操作（管理画面へのアクセス、コンテンツ作成等） | App JWT の `role` クレームに基づきミドルウェアで即時判定 |
| **ACL** | 個別のコンテンツ（ビデオ等）の閲覧・検索 | データベース（MongoDB）のクエリ条件としてフィルタリング |

## 2. ロールと権限定義

### ロール階層
1.  **Admin (管理者)**: システムの全リソースに対するフルコントロール。
2.  **User (一般ユーザー)**: 公開済みコンテンツの閲覧、および許可された制限付きコンテンツの閲覧。

### 具体的なパーミッション例
| 機能区分 | パーミッション名 | User | Admin | 備考 |
| :--- | :--- | :--- | :--- | :--- |
| **コンテンツ管理** | `contents:write` | - | ○ | メタデータの作成・更新・削除 |
| **コンテンツ閲覧** | `contents:read` | ○ | ○ | 映像作品のメタデータ取得 (ACL対象) |
| **カタログ検索** | `catalog:read` | ○ | ○ | 公開用フロントエンドでの検索 (ACL対象) |
| **ユーザー管理** | `users:manage` | - | ○ | ユーザー一覧、ロール変更等 |

## 3. リソース単位のアクセス制御 (ACL)

特定のビデオを「特定のグループのみ」に限定して公開する場合の制御フローです。

### データ構造 (PostgreSQL & MongoDB)
コンテンツドキュメントに以下の属性を持たせます。
- `visibility`: `public` (全員) または `restricted` (限定)
- `allowed_groups`: アクセスを許可するグループ ID のリスト

### 認可検索 (Authorized Search)
Catalog Service はリクエスト元の App JWT から `groups` クレームを取得し、以下の論理で検索を実行します。

```javascript
// MongoDB クエリイメージ
find({
  $or: [
    { visibility: "public" },
    { allowed_groups: { $in: user_groups_from_jwt } }
  ]
})
```

## 4. JWT への権限情報の埋め込み

`auth-service` が発行する App JWT には、認可判定に必要な最小限の情報をペイロードに含めます。

```json
{
  "sub": "user-uuid",
  "email": "user@example.com",
  "role": "admin",
  "groups": ["marketing", "internal-beta"],
  "iat": 1600000000,
  "exp": 1600003600
}
```

## 5. 実装指針

1.  **Backend (Go)**: 
    - [x] JWT の署名検証
    - [x] ロールベースの認可判定ミドルウェア
2.  **Data Sync (Benthos)**:
    - [x] PostgreSQL の権限属性（`visibility`, `allowed_groups`）を MongoDB へ同期
3.  **Frontend**:
    - [ ] App JWT のロールに応じた UI の出し分け（管理メニューの表示等）
