-- contents テーブルに代表サムネイルのS3キーを保持する列を追加する
-- CDC経由でMongoDBへ自動反映させるため、サムネイル情報を assets テーブルではなく
-- contents テーブル自体に持たせる
ALTER TABLE contents ADD COLUMN IF NOT EXISTS thumbnail_key TEXT;
