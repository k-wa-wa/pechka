# NFSビデオインポートバッチ仕様 (NFS Video Importer Batch)

## 1. 概要
NFSマウントされたディレクトリを定期的にスキャンし、新しく追加されたHLS形式のビデオコンテンツ（`.m3u8`ファイル）を自動的に検出し、`Metadata Service`（PostgreSQL）に登録する。

## 2. 処理フロー
1.  **パスのスキャン**: 設定されたNFSマウントパス（例: `/mnt/nfs/videos`）を再帰的にスキャンする。
2.  **HLSの特定**: 拡張子が `.m3u8` のファイルを探す。
3.  **重複チェック**: 検出されたファイルの絶対パス（またはハッシュ）がすでにDBに存在するか確認する。
4.  **MinIOへのアップロード**: 
    -   検出された `.m3u8` および、それに付随する `.ts` セグメントファイルを MinIO（Object Storage）にアップロードする。
    -   アップロード先のパス（S3 Key）は、一意なIDまたはディレクトリ構造に基づき決定する。
    -   例: `videos/{content_id}/master.m3u8`, `videos/{content_id}/segment_001.ts`
5. **メタデータ登録**:
    -   `contents` テーブルに新規レコードを作成（`type='video'`）。
    -   `content_videos` テーブルに動画詳細情報を登録。
    -   `assets` テーブルに以下の 2 種類の情報を登録する：
        -   `minio_key` (Internal): MinIO 内のオブジェクトパス。ライフサイクル管理やバックエンド処理で使用。
        -   `public_url` (External): クライアント（hls.js）がアクセスする URL。CDN (Cloudflare 等) のドメインを含む。
6. **初期値の保持**: タイトルはファイル名、説明文は空文字などで初期登録し、後の管理画面での編集を待つ。

## 3. 技術的詳細
-   **実行形態**: Cron Job または K8s CronJob。
-   **言語**: Go (既存の `stream/api` 内のライブラリや AWS SDK for Go V2 を利用)。
-   **パス解決**:
    -   NFS上の相対パスをそのまま MinIO のキー構造に投影、または UUID を用いて整理する。
-   **アップロード管理**: 大きな動画ファイルに対応するため、マルチパートアップロードの利用や、転送成功時のみDB登録を行うトランザクション管理を行う。

## 4. 検討事項・決定事項
-   **ストレージ参照**: NFS はインポート元としてのみ使用し、配信およびシステムからの参照は **MinIO のみに統一** する。
-   **URL 管理戦略**:
    -   `Asset` レコードは `s3_key` と `external_url` の両方を保持する設計とする。
    -   `s3_key` 例: `videos/123-456/master.m3u8`
    -   `external_url` 例: `https://cdn.example.com/videos/123-456/master.m3u8`
    -   これにより、将来的な CDN 切り替えやドメイン変更に柔軟に対応可能とする。

## 5. 実装時の必要情報
-   **MinIO 接続情報**: Endpoint, AccessKey, SecretKey, BucketName。
-   **CDN ベースURL**: インポート時に `external_url` を生成するためのベースドメイン。
-   **NFS スキャンルート**: インポート対象とする NFS 上のディレクトリパス。
-   **ファイル走査ロジック**: `.m3u8` を見つけた際、同階層にある `.ts` ファイルもセットでアップロード対象とする。
