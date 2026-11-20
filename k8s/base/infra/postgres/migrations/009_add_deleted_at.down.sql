BEGIN;

-- 論理削除後の同一 disc_id 再取り込みが行われていた場合、旧一意制約へ戻すと重複で失敗する。
-- インデックスを削除する前に復元可否を検査し、不可ならトランザクションごとロールバックして現行制約を保全する。
DO $$
DECLARE
    dup_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO dup_count
    FROM (
        SELECT disc_id
        FROM contents
        WHERE disc_id IS NOT NULL
        GROUP BY disc_id
        HAVING COUNT(*) > 1
    ) dups;

    IF dup_count > 0 THEN
        RAISE EXCEPTION 'cannot rollback 009_add_deleted_at: % duplicate disc_id(s) exist including soft-deleted rows', dup_count;
    END IF;
END;
$$;

DROP INDEX IF EXISTS idx_contents_disc_id;
CREATE UNIQUE INDEX IF NOT EXISTS idx_contents_disc_id ON contents(disc_id) WHERE disc_id IS NOT NULL;
DROP INDEX IF EXISTS idx_contents_deleted_at;
ALTER TABLE contents DROP COLUMN IF EXISTS deleted_at;

COMMIT;
