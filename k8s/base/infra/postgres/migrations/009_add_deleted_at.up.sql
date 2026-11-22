ALTER TABLE contents ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- 論理削除済みレコードと同一 disc_id の再取り込みを可能にするため、
-- idx_contents_disc_id を WHERE disc_id IS NOT NULL AND deleted_at IS NULL の部分一意インデックスへ再定義する。
-- 毎回再実行される migrate Job で不要なインデックス再構築を避けるため、未更新時のみ再作成する。
DO $$
DECLARE
    idx_def TEXT;
BEGIN
    SELECT pg_get_indexdef(c.oid)
    INTO idx_def
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = current_schema()
      AND c.relname = 'idx_contents_disc_id'
      AND c.relkind = 'i';

    IF idx_def IS NULL OR LOWER(idx_def) NOT LIKE '%deleted_at is null%' THEN
        DROP INDEX IF EXISTS idx_contents_disc_id;
        CREATE UNIQUE INDEX idx_contents_disc_id
            ON contents(disc_id)
            WHERE disc_id IS NOT NULL AND deleted_at IS NULL;
    END IF;
END;
$$;
