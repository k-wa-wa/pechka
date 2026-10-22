DROP INDEX IF EXISTS idx_contents_channel_id;
ALTER TABLE contents DROP COLUMN IF EXISTS channel_id;
DROP TRIGGER IF EXISTS channels_updated_at ON channels;
DROP TABLE IF EXISTS channels;
