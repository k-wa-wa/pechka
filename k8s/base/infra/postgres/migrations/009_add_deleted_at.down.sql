BEGIN;

-- 論理削除済みレコードが残る状態で deleted_at 列を落とすと、旧 API から再び有効行として公開されてしまう。
-- また、論理削除後の同一 disc_id 再取り込みが行われていた場合、旧一意制約へ戻すと重複で失敗する。
-- いずれかに該当する場合は列やインデックスを変更する前にトランザクションごとロールバックする。
DO $$
DECLARE
    deleted_count INTEGER;
    dup_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO deleted_count
    FROM contents
    WHERE deleted_at IS NOT NULL;

    IF deleted_count > 0 THEN
        RAISE EXCEPTION 'cannot rollback 009_add_deleted_at: % soft-deleted row(s) exist in contents', deleted_count;
    END IF;

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
ALTER TABLE contents DROP COLUMN IF EXISTS deleted_at;

COMMIT;
