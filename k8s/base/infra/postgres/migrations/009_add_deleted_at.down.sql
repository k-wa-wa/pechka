DROP INDEX IF EXISTS idx_contents_disc_id;
CREATE UNIQUE INDEX IF NOT EXISTS idx_contents_disc_id ON contents(disc_id) WHERE disc_id IS NOT NULL;
DROP INDEX IF EXISTS idx_contents_deleted_at;
ALTER TABLE contents DROP COLUMN IF EXISTS deleted_at;
