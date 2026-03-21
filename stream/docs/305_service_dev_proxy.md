# 305 Dev Proxy Service (開発用プロキシ & 認証モック)

ローカル開発環境（Kind）において、Cloudflare Access などの外部認証プロバイダーをシミュレートし、JWKS による公開鍵配布と JWT 検証を可能にするための統合サービスです。また、ゲートウェイ（Cilium Gateway）へのリバースプロキシとしても機能し、未認証アクセスに対するログイン画面の提供も行います。

## 構成と役割

- **Location**: `tests/dev-proxy` に独立した Go モジュールとして配置されています。
- **JWKS Endpoint**: `/.well-known/jwks.json` を公開し、検証用の固定 RSA 公開鍵を提供します。
- **Token Issuer**: `POST /mock/token` エンドポイントを通じて、任意のメールアドレスを含む署名済み JWT を発行します（テストコード用）。
- **Interactive Auth**: `/` などのプロキシ対象パスに未認証でアクセスした場合、`/dev-proxy/login`（メールアドレス入力画面）へリダイレクトします。
- **Reverse Proxy**: 環境変数 `PROXY_TARGET`（デフォルトは Cilium Gateway）へリクエストを転送します。ホスト側からは NodePort 30000 経由でアクセス可能です。

## 認証フロー

### ブラウザ経由（インタラクティブ）
1.  **アクセス**: ブラウザで `http://localhost:8000/` にアクセス。
2.  **リダイレクト**: `dev-proxy` が Cookie (`cf-access-token-mock`) の不在を検知し、ログイン画面を表示。
3.  **ログイン**: メールアドレスを入力して送信すると、`dev-proxy` が JWT を発行し Cookie にセット。
4.  **プロキシ**: 元の URL へリダイレクト。`dev-proxy` は Cookie 内の JWT を `Cf-Access-Jwt-Assertion` ヘッダーとして付与し、バックエンドへ転送。

### プログラム・テスト経由
1.  **Cookieの付与**: `cf-access-token-mock` Cookie に有効な JWT をセットしてリクエストを送信すると、`dev-proxy` がそれを検知して `Cf-Access-Jwt-Assertion` ヘッダーを付与し、バックエンドへ転送します。
2.  **未認証時の挙動**: Cookie が存在しない場合、API 呼び出しであってもブラウザ同様に `/dev-proxy/login` へのリダイレクト（302）が発生します。これは実際の Cloudflare Access のデフォルト挙動を模したものです。

## 設定項目 (ConfigMap)

| 変数名 | 説明 | ローカル設定値例 |
| :--- | :--- | :--- |
| `CLOUDFLARE_JWKS_URL` | JWKSを取得するURL | `http://dev-proxy:8080/.well-known/jwks.json` |
| `CLOUDFLARE_JWT_ISSUER` | JWTの `iss` クレーム期待値 | `https://pechka.cloudflareaccess.com` |
| `CLOUDFLARE_JWT_AUDIENCE` | JWTの `aud` クレーム期待値 | `mock-audience` |
| `PROXY_TARGET` | プロキシ先の内部URL | `http://cilium-gateway-cilium-gateway.default.svc.cluster.local:80` |

## 開発作業用 Makefile コマンド
- `make build-dev-proxy`: イメージのビルド
- `make load-dev-proxy`: Kind へのイメージロード
- `make reload-dev-proxy`: ビルド、ロード、および Deployment の再起動を一括実行
