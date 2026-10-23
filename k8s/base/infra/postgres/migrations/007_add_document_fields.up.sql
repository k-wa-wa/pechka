-- ドキュメントコンテンツ(.md / .pdf 等)の配信キーとMIMEタイプを保持する列を追加する
-- thumbnail_key と同様、Benthos CDC 経由で MongoDB / Elasticsearch へ直接同期させるため
-- contents テーブル自体に持たせる
ALTER TABLE contents ADD COLUMN IF NOT EXISTS document_key TEXT;
ALTER TABLE contents ADD COLUMN IF NOT EXISTS document_mime_type VARCHAR(100);
