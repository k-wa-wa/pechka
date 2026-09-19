# pechka

動画・画像・VR・ドキュメント等、あらゆる種類のコンテンツを NAS やディスク等から取り込み、ブラウザで閲覧・配信するホームメディア基盤。Bluray に限らず様々なデータソース（NAS、ファイルサーバ、各種メディア）への対応を視野に入れ、コンテンツを AI へのインプットとして活用することも中核的なユースケースとして想定している。

## 概要

既存のホームオートメーション関連コードを整理し、以下の構成で再構築しています。
詳細は `docs/` フォルダを参照してください。

## ドキュメント

- [101_requirements.md](docs/101_requirements.md) — 要件定義書
- [102_tech_digest_user_stories.md](docs/102_tech_digest_user_stories.md) — 技術ダイジェスト機能 ユーザーストーリー
- [201_architecture.md](docs/201_architecture.md) — アーキテクチャ設計書
- [405_bluray_ingestion_pipeline.md](docs/405_bluray_ingestion_pipeline.md) — Bluray ETL パイプライン設計
- [406_subtitle_generation_pipeline.md](docs/406_subtitle_generation_pipeline.md) — ライブ字幕自動生成パイプライン設計
- [407_tech_digest_pipeline.md](docs/407_tech_digest_pipeline.md) — 技術ダイジェスト自動生成パイプライン 技術調査・構成検討

## 現在のステータス

主要コンポーネントは実装済みで、`k8s/overlays/prod` 上で稼働している。

- **インフラ・DB 基盤**: PostgreSQL（書き込み） / MongoDB（読み取り最適化） / Elasticsearch（全文検索） / Benthos（PostgreSQL → MongoDB・Elasticsearch への CDC 同期）— 実装済み
- **API Service（Go）**: 実装済み（`api/`）
- **フロントエンド（Next.js）**: 実装済み（`frontend/`）
- **Bluray ETL パイプライン**: 実装済み（`batch/etl/`）。現状は NFS 上の MKV を対象とした手動実行（Argo Workflows `etl-bluray` の `manual` エントリーポイント）のみで、物理ドライブの自動監視ジョブは未整備
- **字幕自動生成パイプライン**: 実装済み（`batch/subtitle/`、Argo Workflows `subtitle-gen`）
- **技術ダイジェスト自動生成パイプライン（tech-feed）**: 実装済み（`batch-tech-feed/`）。詳細は後述

今後の計画（ドキュメント等の多様コンテンツ取り込み、AI インプット活用等）は [201_architecture.md](docs/201_architecture.md) の実装フェーズ計画を参照。

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

### PR プレビュー環境

各 PR には ArgoCD により専用の preview 環境（`k8s/overlays/preview`）がデプロイされ、アプリ本体に加えて
Storybook のビルド成果物・VRT（Playwright）レポートもあわせて配信されます。

- アプリ本体: `/`
- Storybook 静的ビルド: `/storybook`（`storybook-report/`。CI が `storybook-static` アーティファクトを配信用コンテナイメージ化）
- VRT レポート: `/vrt-report`（`vrt-report/`。CI が `playwright-report` アーティファクトを配信用コンテナイメージ化）

---

## 技術ダイジェスト自動生成パイプライン（batch-tech-feed）

`batch-tech-feed/` は、技術に関する最新情報を収集し、同一台本から解説動画と記事を自動生成して
pechka 上のコンテンツとして配信するパイプラインです。

- `collect/`（Go）: 情報源からの収集・選定・一次情報検証・台本生成
- `produce/`（Go）: 台本からの音声合成・動画レンダリング・記事生成・pechka への配信

Argo CronWorkflow（`k8s/base/tech-feed/`）により定期実行されます。AI 分野向け（日次、稼働中）と
k8s 分野向け（週次、現状一時停止中）の 2 系統があります。詳細は
[102_tech_digest_user_stories.md](docs/102_tech_digest_user_stories.md)（ユーザーストーリー）・
[407_tech_digest_pipeline.md](docs/407_tech_digest_pipeline.md)（技術調査・構成検討、未レビュー）を参照してください。

---

## 本番環境（overlays/prod）のデプロイと運用

本番環境向けのマニフェストは `k8s/overlays/prod` に整理されています。

### 1. データベースおよびオブジェクトストレージ
- PostgreSQL・MinIO は nuage-cluster リポジトリが管理する外部インスタンスを使用します。`external-postgres.yaml` /
  `external-minio.yaml` が `ExternalName` Service として同名ホストへ転送するだけで、実 IP はこのリポジトリ側では持ちません。
- MongoDB・Elasticsearch は `k8s/overlays/prod` 内で稼働するコンテナです（PVC は `local-path` storageClass、
  `pvc-local-path-patch.yaml` で上書き）。

### 2. ETL バッチ処理の実行
- NFS 上の MKV ファイルを対象とした Bluray ETL は、WorkflowTemplate `etl-bluray` の `manual` エントリーポイントから
  Argo Web UI や CLI で手動実行します。物理ドライブを監視して自動変換を行うスケジュールバッチは現状未整備です。
- 字幕自動生成（`subtitle-gen`）も同様に手動実行の WorkflowTemplate です。
- 技術ダイジェスト自動生成（tech-feed、前述）のみ Argo CronWorkflow による定期自動実行です。

### 3. 秘密情報の管理（SOPS）
- `k8s/overlays/prod/secrets/prod-secrets.yaml` に本番の接続情報（DB / MinIO / MongoDB / 各種 API キー等）を定義し、
  `sops` で age 鍵暗号化した上でリポジトリにコミットしています。
- `k8s/overlays/prod/secrets/secrets.yaml` が `<path:secrets/prod-secrets.yaml#KEY>` というプレースホルダー参照で
  各 Secret リソースを組み立て、ArgoCD 側の `argocd-vault-plugin` が Sync 時に実値へ復号します。

