# Auth Service 設計詳細

## 1. サービスの目的と責務
エンドユーザーの認証、登録、セッション管理（JWT）を単一のサービスとして切り出す。CatalogやMetadataなどの他サービスは、Auth Serviceが発行したJWTを検証することでリクエストの正当性を判断し、直接ユーザーDBへはアクセスしない。

## 2. API エンドポイント定義

## 2. API エンドポイント定義

### `GET /api/v1/auth/session`
- **目的**: 外部認証（Cloudflare Access または Dev Proxy）のトークンをアプリ専用の JWT (App JWT) へ交換する。
- **Header**: `Cf-Access-Jwt-Assertion` (Cloudflare が付与する JWT、または Dev Proxy が付与するモック JWT)
- **Response**: `access_token` (App JWT)
- **重要事項**: 
  - このエンドポイントは **クライアントサイド（ブラウザ）から直接呼び出すこと**。
  - 内部では `Cf-Access-Jwt-Assertion` を検証し、ユーザーが存在しない場合は自動的に `users` テーブルへ作成（JITプロビジョニング）を行う。
  - 成功時、フロントエンドは取得した `access_token` をローカルストレージ等に保持し、以降のリクエスト（`Authorization: Bearer ...`）に使用する。

### `GET /api/v1/auth/me`
- **目的**: 現在のログインユーザー情報（ロール、グループ、権限を含む）を取得する。
- **Header**: `Authorization: Bearer <App JWT>`
- **Response**:
  ```json
  {
    "id": "e02b... (UUID)",
    "email": "user@example.com",
    "displayName": "User Name",
    "roles": ["admin"],
    "groups": ["Administrators"],
    "permissions": ["content:read", "content:write"]
  }
  ```

## 3. JWT 戦略
- **Access Token**: 寿命を短く（例: 15分〜1時間）設定。ペイロードには `user_id` (UUID), `role` (Admin/User), などの基本情報を持たせ、各サービス(Go)側で秘密鍵/公開鍵を用いて署名検証のみで完結させる（通信不要）。
- **Refresh Token**: 寿命を長く（例: 14日）設定。データベース（PostgreSQL）でハッシュ化して管理し、明示的なログアウト時にRevoke可能にする。HttpOnly Cookieに格納してXSS攻击を防ぐ。
