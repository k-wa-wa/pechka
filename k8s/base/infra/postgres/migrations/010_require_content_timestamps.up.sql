-- 既存の NULL created_at / updated_at を補完し、NOT NULL 制約を付与して MongoDB 移行判定・フォールバックスキャン失敗を防ぐ。
UPDATE contents
SET created_at = COALESCE(created_at, updated_at, published_at, archived_at, CURRENT_TIMESTAMP),
    updated_at = COALESCE(updated_at, created_at, published_at, archived_at, CURRENT_TIMESTAMP)
WHERE created_at IS NULL OR updated_at IS NULL;

DO $$
BEGIN
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
