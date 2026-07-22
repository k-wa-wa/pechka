DROP INDEX IF EXISTS idx_contents_disc_id;
ALTER TABLE video_variants DROP CONSTRAINT IF EXISTS unique_video_variants_content_variant;
ALTER TABLE assets DROP CONSTRAINT IF EXISTS unique_assets_content_role_key;
