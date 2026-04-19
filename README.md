# pechka

動画・画像・VR・ドキュメント等、あらゆる種類のコンテンツを NAS やディスク等から取り込み、ブラウザで閲覧・配信するホームメディア基盤。Bluray に限らず様々なデータソース（NAS、ファイルサーバ、各種メディア）への対応を視野に入れ、コンテンツを AI へのインプットとして活用することも中核的なユースケースとして想定している。

## 概要

既存のホームオートメーション関連コードを整理し、以下の構成で再構築しています。
詳細は \`docs/\` フォルダを参照してください。

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
