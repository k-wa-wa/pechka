# pechka

動画・画像・VR・ドキュメント等、あらゆる種類のコンテンツを NAS やディスク等から取り込み、ブラウザで閲覧・配信するホームメディア基盤。Bluray に限らず様々なデータソース（NAS、ファイルサーバ、各種メディア）への対応を視野に入れ、コンテンツを AI へのインプットとして活用することも中核的なユースケースとして想定している。

## 概要

既存のホームオートメーション関連コードを整理し、以下の構成で再構築しています。
詳細は `docs/` フォルダを参照してください。

## ドキュメント

- [101_requirements.md](docs/101_requirements.md) — 要件定義書
- [201_architecture.md](docs/201_architecture.md) — アーキテクチャ設計書
- [405_bluray_ingestion_pipeline.md](docs/405_bluray_ingestion_pipeline.md) — Bluray ETL パイプライン設計

## 現在のステータス

- **Phase 0: クリーンアップ + ドキュメント整備** (進行中 - PR #3)
- **Phase 1: インフラ・DB 基盤** (準備中)
- **Phase 2: API Service 実装 (Go)** (準備中)
- **Phase 3: フロントエンド実装 (Next.js)** (準備中)
- **Phase 4: Bluray ETL パイプライン実装** (準備中)
- **Phase 5: 最適化・運用整備** (準備中)

## フロントエンドのみの UI 開発（API・K8s なし）

コンテンツの一覧・表示崩れの調整など UI 中心の変更であれば、K8s 環境や実 API を
起動せずに `frontend/` だけで作業できます。

### Storybook でコンポーネント単位に確認する

```bash
cd frontend
npm run storybook
```

[http://localhost:6006](http://localhost:6006) で各コンポーネントを単体表示できます。API 通信は
[msw-storybook-addon](https://github.com/mswjs/msw-storybook-addon) が `mocks/handlers.ts` の
ハンドラで自動的にモックするため、バックエンドは不要です。ローディング/空/エラー状態など
個別のシナリオは各 `*.stories.tsx` の `parameters.msw.handlers` でハンドラを上書きして表現します。

### アプリ全体をモック API 付きで動かす

ページ遷移やアプリ全体の見た目を確認したい場合は、`e2e/mock-server.mjs`（Playwright の VRT
テストでも使用している Node 製モック API サーバ）を使って `next dev` を起動します。

```bash
cd frontend
npm run dev:mock
```

[http://localhost:3000](http://localhost:3000) にアクセスすると、モックデータでアプリ全体（一覧・
詳細・管理画面）が動作します。

### 新しいコンポーネント・API を追加したとき

- `components/` に新しいコンポーネントを追加したら、同名の `*.stories.tsx` を追加してください。
  CI (`npm run check:stories`) が漏れを検知して失敗します。
- `lib/api.ts` に新しいエンドポイントを追加したら、`mocks/handlers.ts`（Storybook 用）と
  `e2e/mock-server.mjs`（`dev:mock`・VRT 用）の両方にモックハンドラを追加してください。

---

## 本番環境（overlays/prod）のデプロイと運用

本番環境向けのマニフェストは `k8s/overlays/prod` に整理されています。現状は検証用として、以下の構成となっています。

### 1. NFS 接続
NFS サーバー（`10.20.1.30`）の各ディレクトリをマウントします。
- 一旦自動 Bluray ディスク変換を行わない期間中は、安全のため NFS PV および PVC の接続モードはすべて `ReadOnlyMany`（読み取り専用）に制限されています。

### 2. ETL バッチ処理の実行
- 物理ドライブを監視して自動で Bluray 変換を行うスケジュールバッチ（CronWorkflow `etl-bluray-cron`）は、本番パッチ（`workflow-patch.yaml`）によって `suspend: true`（一時停止）に設定されています。
- すでにディスクから抽出済みの MKV ファイルを NFS 上からスキャンして処理する手動実行バッチ（WorkflowTemplate `etl-bluray` の `manual` エントリーポイント）は、Argo Web UI や CLI から手動実行が可能です。

### 3. データベースおよびオブジェクトストレージ
- 現状の NFS データで手軽に動作検証が行えるよう、検証中は PostgreSQL と MinIO も一時的なコンテナとして同一クラスター内に起動するように設定されています（`tmp/` 配下に定義）。
- 将来的に外部の PostgreSQL や AWS S3 などの外部オブジェクトストレージに切り替える際は、 `k8s/overlays/prod/kustomization.yaml` から `tmp/postgres` および `tmp/minio` のリソース参照を削除するだけで切り替えが可能です。

### 4. 秘密情報の管理（SOPS）
- 現在は一時的な検証用として、 `k8s/overlays/prod/secrets.yaml` 内に一時コンテナ向けのテスト用 ID/PW がハードコードされています。
- 本番の外部DB接続へ移行する際は、 `k8s/overlays/prod/secrets/prod-secrets.yaml` に実際の接続情報を定義し、 `sops` コマンド等で暗号化した上で、 `secrets.yaml` のプレースホルダー参照を本番用の実定義に差し替えて運用してください。

