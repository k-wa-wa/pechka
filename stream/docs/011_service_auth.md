# Auth Service 設計詳細 (service_auth.md)

## 1. サービスの目的と責務
エンドユーザーの認証、登録、セッション管理（JWT）を単一のサービスとして切り出す。CatalogやMetadataなどの他サービスは、Auth Serviceが発行したJWTを検証することでリクエストの正当性を判断し、直接ユーザーDBへはアクセスしない。

## 2. API エンドポイント定義

### `POST /api/v1/auth/register`
- **目的**: 新規ユーザー登録
- **Payload**:
  ```json
  {
    "email": "user@example.com",
    "password": "strongPassword123"
  }
  ```
- **Response**: JWT access token, refresh token (HttpOnly Cookie) を返却。
- **処理**: bcryptでパスワードをハッシュ化し、PostgreSQLの `users` テーブルへ保存。

### `POST /api/v1/auth/login`
- **目的**: ログイン
- **Payload**: Registerと同様。
- **Response**: 成功時にJWTを返却。

### `POST /api/v1/auth/refresh`
- **目的**: Access Tokenの再発行
- **Header**: Refresh Token (Cookie)
- **Response**: 新しい Access Token を返却。

### `GET /api/v1/auth/me`
- **目的**: 現在のログインユーザー情報取得
- **Header**: `Authorization: Bearer <JWT>`
- **Response**:
  ```json
  {
    "id": "e02b... (UUID)",
    "email": "user@example.com",
    "role": "user"
  }
  ```

## 3. JWT 戦略
- **Access Token**: 寿命を短く（例: 15分〜1時間）設定。ペイロードには `user_id` (UUID), `role` (Admin/User), などの基本情報を持たせ、各サービス(Go)側で秘密鍵/公開鍵を用いて署名検証のみで完結させる（通信不要）。
- **Refresh Token**: 寿命を長く（例: 14日）設定。データベース（PostgreSQL）でハッシュ化して管理し、明示的なログアウト時にRevoke可能にする。HttpOnly Cookieに格納してXSS攻击を防ぐ。
