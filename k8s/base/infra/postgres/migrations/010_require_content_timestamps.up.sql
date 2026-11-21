-- 既存の NULL created_at / updated_at を補完し、NOT NULL 制約を付与して MongoDB 移行判定・フォールバックスキャン失敗を防ぐ。
-- 片方だけが NULL の行を更新する際に BEFORE UPDATE トリガー（contents_updated_at）が元の updated_at を移行時刻で上書きしないよう、
-- 同一トランザクション内で対象トリガーを一時無効化して補完後に復元する。
BEGIN;

DO $$
DECLARE
    has_trigger BOOLEAN;
BEGIN
    IF EXISTS (SELECT 1 FROM contents WHERE created_at IS NULL OR updated_at IS NULL) THEN
        SELECT EXISTS (
            SELECT 1
            FROM pg_trigger
            WHERE tgrelid = 'contents'::regclass
              AND tgname = 'contents_updated_at'
              AND tgenabled <> 'D'
        ) INTO has_trigger;

        IF has_trigger THEN
            EXECUTE 'ALTER TABLE contents DISABLE TRIGGER contents_updated_at';
        END IF;

        UPDATE contents
        SET created_at = COALESCE(created_at, updated_at, published_at, archived_at, CURRENT_TIMESTAMP),
            updated_at = COALESCE(updated_at, created_at, published_at, archived_at, CURRENT_TIMESTAMP)
        WHERE created_at IS NULL OR updated_at IS NULL;

        IF has_trigger THEN
            EXECUTE 'ALTER TABLE contents ENABLE TRIGGER contents_updated_at';
        END IF;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'contents'
          AND column_name = 'created_at'
          AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE contents ALTER COLUMN created_at SET NOT NULL;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'contents'
          AND column_name = 'updated_at'
          AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE contents ALTER COLUMN updated_at SET NOT NULL;
    END IF;
END;
$$;

COMMIT;
