-- 1. contents テーブルの既存重複データをクリーンアップ（過去の一回限りデータ掃除のためコメントアウト済み）
-- DELETE FROM contents a
-- USING contents b
-- WHERE a.disc_id IS NOT NULL
--   AND a.disc_id = b.disc_id
--   AND a.created_at < b.created_at;

-- contents の disc_id に一意インデックスを追加（009 で deleted_at IS NULL 付きに更新済みの場合は上書きしない）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema()
          AND c.relname = 'idx_contents_disc_id'
          AND c.relkind = 'i'
    ) THEN
        CREATE UNIQUE INDEX idx_contents_disc_id ON contents(disc_id) WHERE disc_id IS NOT NULL;
    END IF;
END;
$$;

-- 2. video_variants テーブルの既存重複データをクリーンアップ（過去の一回限りデータ掃除のためコメントアウト済み）
-- DELETE FROM video_variants a
-- USING video_variants b
-- WHERE a.content_id = b.content_id
--   AND a.variant_type = b.variant_type
--   AND a.created_at < b.created_at;

-- video_variants の (content_id, variant_type) に一意制約を追加
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'unique_video_variants_content_variant') THEN
        ALTER TABLE video_variants ADD CONSTRAINT unique_video_variants_content_variant UNIQUE (content_id, variant_type);
    END IF;
END;
$$;

-- 3. assets テーブルの既存重複データをクリーンアップ（過去の一回限りデータ掃除のためコメントアウト済み）
-- DELETE FROM assets a
-- USING assets b
-- WHERE a.content_id = b.content_id
--   AND a.asset_role = b.asset_role
--   AND a.s3_key = b.s3_key
--   AND a.created_at < b.created_at;

-- assets の (content_id, asset_role, s3_key) に一意制約を追加
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'unique_assets_content_role_key') THEN
        ALTER TABLE assets ADD CONSTRAINT unique_assets_content_role_key UNIQUE (content_id, asset_role, s3_key);
    END IF;
END;
$$;
