# Admin コンテンツ権限管理 仕様

## 1. 概要

管理画面 (`/admin`) にて、各コンテンツのアクセス権限（公開範囲）を編集できるようにする。

`contents` テーブルには既に `visibility` と `allowed_groups` カラムが存在しており、Catalog Service の検索・閲覧フィルタリングに使用されている。本機能ではこれらのフィールドを管理画面から編集可能にする。

---

## 2. 権限モデル

```
visibility: "public" | "group_only"
allowed_groups: UUID[]  ← groups.id の配列
```

| visibility | 意味 |
| :--- | :--- |
| `public` | 全ユーザーが閲覧可能 |
| `group_only` | `allowed_groups` に含まれるグループのメンバーのみ閲覧可能 |

Catalog Service (MongoDB/Elasticsearch) は以下の条件でフィルタリングを行う：

```
(visibility == "public") OR (allowed_groups ∩ user.groups ≠ ∅)
```

---

## 3. 管理画面 UI 仕様

### 3.1 コンテンツ一覧テーブル

各行に `visibility` の状態を小バッジで表示する：
- `public` → 緑バッジ "Public"
- `group_only` → 黄バッジ "Group Only"

### 3.2 編集モーダル - Access Control セクション

編集モーダル内に **Access Control** セクションを追加する。

#### Visibility 選択 (トグルボタン)

```
[ Public ]  [ Group Only ]
```

- `Public` を選択した場合: `allowed_groups` を空にリセット
- `Group Only` を選択した場合: グループ選択UIを表示

#### グループ選択 (Group Only 時のみ表示)

- バックエンドの `GET /admin/metadata/groups` から全グループ一覧を取得
- グループをボタングリッドで表示（選択済み = 赤ハイライト）
- 複数グループの選択が可能

---

## 4. API 変更

### 4.1 `PUT /api/metadata/v1/admin/metadata/contents/:id`

リクエストボディに `visibility` と `allowed_groups` を追加。

```json
{
  "title": "...",
  "description": "...",
  "rating": 8.5,
  "tags": ["action"],
  "visibility": "group_only",
  "allowed_groups": ["550e8400-e29b-41d4-a716-446655441001"]
}
```

| フィールド | 型 | 必須 | デフォルト |
| :--- | :--- | :--- | :--- |
| `visibility` | `string` | No | `"public"` |
| `allowed_groups` | `string[]` (UUID) | No | `[]` |

### 4.2 `GET /api/metadata/v1/admin/metadata/groups` (新規追加)

グループ一覧を返す。フロントエンドがグループ選択UIを構築するために使用する。

**Response:**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Administrators",
    "description": "System administrators group",
    "created_at": "2025-01-01T00:00:00Z"
  },
  {
    "id": "550e8400-e29b-41d4-a716-446655441001",
    "name": "nfs-admin",
    "description": "NFS Video Administrators",
    "created_at": "2025-01-01T00:00:00Z"
  }
]
```

**認可**: `content:write` 権限または `admin` ロールが必要（既存の Metadata Service ミドルウェアにより適用済み）。

---

## 5. nfs-admin 管理画面アクセス修正

### 5.1 問題の特定

nfs-admin グループのユーザーが管理画面にアクセスしてもコンテンツが表示されない原因：

1. **DB**: `nfs-admin` グループにロール/パーミッションが割り当てられていなかった → API が 403 を返す
2. **Frontend**: `admin/page.tsx` が `fetch()` を直接使用しており JWT トークンが付与されなかった → API が 401 を返す
3. **Navbar**: `admin` ロール保持者のみに管理ダッシュボードリンクを表示していた

### 5.2 修正内容

#### DB (`01_init.sql`)

`content-editor` ロールを追加し、`nfs-admin` グループに割り当て：

```sql
INSERT INTO roles (name, description) VALUES ('content-editor', 'Content creation and editing access');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'content-editor' AND p.resource = 'content';

INSERT INTO group_roles (group_id, role_id)
SELECT g.id, r.id FROM groups g, roles r
WHERE g.name = 'nfs-admin' AND r.name = 'content-editor';
```

これにより nfs-admin は `content:read` および `content:write` の権限を持つ。

#### Frontend - `admin/page.tsx`

`fetch()` → `apiClient` (axios) に変更。`apiClient` は localStorage の JWT トークンを自動的に `Authorization: Bearer <token>` として付与する。

#### Frontend - `Navbar.tsx`

管理ダッシュボードリンクの表示条件を変更：

```typescript
// 変更前
user.roles.includes('admin')

// 変更後
user.roles.includes('admin') || user.permissions.includes('content:write')
```

---

## 6. ロール設計 (更新後)

| ロール | パーミッション | 割り当てグループ |
| :--- | :--- | :--- |
| `admin` | 全権限 | Administrators |
| `content-editor` | `content:read`, `content:write` | nfs-admin |
| `viewer` | `content:read` | - |

---

## 7. データフロー (権限変更時)

```
Admin UI (ブラウザ)
  └─ PUT /admin/metadata/contents/:id  { visibility, allowed_groups }
       └─ Metadata Service (PostgreSQL 更新)
            └─ updated_at トリガー
                 └─ Benthos 自動同期 (10秒間隔)
                      ├─ MongoDB (Catalog) 更新
                      └─ Elasticsearch 更新
```

Benthos の `content_sync_view` は `allowed_groups` を UUID から group name に変換してから MongoDB/ES に同期するため、Catalog Service は group name ベースでフィルタリングを行う。
