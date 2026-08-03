DROP INDEX IF EXISTS idx_contents_archived_at;
ALTER TABLE contents DROP COLUMN IF EXISTS archived_at;
