# テストユーザー一覧

`db/02_seed_test.sql` で定義されているテストユーザーの説明です。
`make up` 時に自動投入されます。

---

## ユーザー一覧

| ユーザー名 | メールアドレス | グループ | ロール | 権限 |
|-----------|--------------|---------|--------|------|
| sys-admin | sys-admin@example.com | Administrators | admin (グループ経由) | 全権限 |
| nfs-editor | nfs-editor@example.com | nfs-admin | content-editor (グループ経由) | content:read, content:write |
| nfs-viewer | nfs-viewer@example.com | なし | viewer (直接付与) | content:read |
| outsider | outsider@example.com | なし | なし | なし |

---

## 各ユーザーの詳細

### sys-admin

- **UUID**: `550e8400-e29b-41d4-a716-446655440010`
- **用途**: システム全体の管理者テスト
- **権限の取得経路**: `Administrators` グループ → `admin` ロール → 全権限
- **できること**:
  - 管理画面 (`/admin`) へのアクセス
  - コンテンツの作成・編集・削除
  - visibility / allowed_groups の変更
  - カタログへの同期
  - グループ一覧の取得 (`GET /admin/metadata/groups`)
  - group_only コンテンツの閲覧（管理 API 経由、ACL フィルタなし）
- **できないこと**: なし（全権限）

---

### nfs-editor

- **UUID**: `550e8400-e29b-41d4-a716-446655440011`
- **用途**: コンテンツ編集者のテスト（管理画面にアクセスできるが、admin ではない）
- **権限の取得経路**: `nfs-admin` グループ → `content-editor` ロール → `content:read` + `content:write`
- **できること**:
  - 管理画面 (`/admin`) へのアクセス（`content:write` 権限により Navbar にリンクが表示される）
  - コンテンツ一覧・詳細の閲覧
  - visibility / allowed_groups の変更
  - `nfs-admin` グループが `allowed_groups` に含まれる `group_only` コンテンツの閲覧（カタログ API）
  - グループ一覧の取得
- **できないこと**:
  - `nfs-admin` グループ以外の `group_only` コンテンツの閲覧

---

### nfs-viewer

- **UUID**: `550e8400-e29b-41d4-a716-446655440012`
- **用途**: 閲覧専用ユーザーのテスト（管理画面に入れない一般ユーザー）
- **権限の取得経路**: `viewer` ロールを直接付与 → `content:read` のみ
- **できること**:
  - `public` コンテンツの閲覧（カタログ API）
- **できないこと**:
  - 管理画面へのアクセス（`content:write` がないため Navbar にリンクが表示されない）
  - 管理 API (`/admin/metadata/...`) の呼び出し → **403**
  - `group_only` コンテンツの閲覧（グループ非所属）→ カタログ API で **404**
  - コンテンツの作成・編集

---

### outsider

- **UUID**: `550e8400-e29b-41d4-a716-446655440013`
- **用途**: 権限なしユーザーのテスト（JIT プロビジョニングされたが何も割り当てられていないケース）
- **権限の取得経路**: なし
- **できること**:
  - ログイン自体は可能（CF Access を通過できる）
- **できないこと**:
  - 管理画面へのアクセス
  - 管理 API の呼び出し → **403**
  - コンテンツの作成・編集
  - ※ カタログ API 側の認可設計による（現状は `content:read` チェックなし）

---

## 認証フロー（テスト環境）

```
loginAs('nfs-editor')
  → POST /mock/token  (dev-proxy, メールアドレスから CF JWT を生成)
  → GET  /api/v1/auth/session  (Cookie: CF JWT → App JWT を返却)
  → App JWT に groups / roles / permissions が含まれる
```

テストヘルパーは `tests/api/helpers/auth.ts` に定義されています。

- `fullHeaders(creds)` → `Authorization: Bearer <App JWT>` + CF Cookie（API テスト用）
- `cookieOnlyHeaders(creds)` → CF Cookie のみ（Bearer なし）→ API 側で **401** になることを確認するケースで使用

---

## グループ・ロールの固定 UUID

| 名前 | UUID |
|------|------|
| nfs-admin グループ | `550e8400-e29b-41d4-a716-446655441001` |

`01_init.sql` で定義されており、テストコードにハードコードされています。
