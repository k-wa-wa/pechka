# 開発環境のセットアップと運用

本プロジェクトでは、開発環境として **kind (Kubernetes in Docker)** を使用します。
構成管理には **Kustomize** を採用しており、環境ごとの差異（開発用、本番用）を `overlays` で管理しています。

## 構成

- `k8s/base/`: 全環境共通のアプリケーション定義
- `k8s/infra/`: 開発用のミドルウェア定義（Postgres, MongoDB, MinIO, etc.）
- `k8s/overlays/dev/`: kind環境固有の設定（ポートフォワード、開発用ボリュームマウント等）

## クイックスタート

Docker Composeからの移行後は、`Makefile` を使用して簡単に環境を立ち上げることができます。

### 初回起動 / 全リセット

```bash
# クラスターの作成、イメージビルド、デプロイを一括実行
make up
```

### 終了

```bash
# クラスターを削除し、すべてのデータを消去します
make down
```

## 日常の開発フロー

### コード変更の反映

変更したコンポーネントに応じて、以下のコマンドを実行します。

| 対象 | コマンド |
| :--- | :--- |
| **すべて反映** | `make reload` |
| **APIサービス** | `make reload-app` |
| **フロントエンド** | `make reload-frontend` |
| **サムネイル解析** | `make reload-analyzer` |
| **バッチ処理の再実行** | `make reload-batch` |

### 高速なフロントエンド開発 (Next.js Hot Reload)

k8s上のフロントエンドはコンテナイメージのビルドとデプロイが必要なため、UIの調整には向きません。ローカルマシン上で `next dev` を起動して開発することを推奨します。

1. **セットアップ**:
   `frontend/` ディレクトリに `.env.local` を作成し、k8s上のGatewayを向くように設定します。
   ```env
   INTERNAL_API_URL=http://localhost:8000
   ```

2. **起動**:
   ```bash
   cd frontend
   npm run dev
   ```
   - `-H 0.0.0.0` で起動するため、同一Wi-Fi内のスマホ等からも `http://[PCのIP]:3001` でアクセス可能です。

### ログの確認

Podの状態とログを確認するには以下のコマンドを使用します。

```bash
# Podの一覧を表示
kubectl get pods

# 特定のPodのログを表示
kubectl logs -f deployment/metadata-service
```

### インフラへの直接アクセス

- **Nginx (Gateway)**: `http://localhost:8000`
- **Frontend (k8s)**: `http://localhost:3000`
- **Frontend (Local Dev)**: `http://localhost:3001`

## トラブルシューティング

### 設定ファイルの変更が反映されない
`nginx.conf` や `init.sql` を変更した場合は、`make sync` を実行するか、`make up` / `make reload` を実行してください。これらは `Makefile` 内で自動的に `k8s/` ディレクトリへ同期されるようになっています。
