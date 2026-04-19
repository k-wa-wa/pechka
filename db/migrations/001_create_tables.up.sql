-- =====================
-- Bluray ディスク管理
-- =====================
CREATE TABLE discs (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    label      VARCHAR(255) NOT NULL UNIQUE,  -- ディスクラベル (blkid から取得)
    disc_name  VARCHAR(255),                  -- 管理画面で設定する名前
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =====================
-- コンテンツ（各タイトル）
-- =====================
CREATE TABLE contents (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    short_id         VARCHAR(50)  UNIQUE NOT NULL,  -- Snowflake ID（外部公開用）
    content_type     VARCHAR(20)  NOT NULL,          -- 'video', 'image_gallery', 'vr360', 'document'
    disc_id          UUID         REFERENCES discs(id),  -- NULL = 手動登録
    title            VARCHAR(255) NOT NULL,
    description      TEXT         DEFAULT '',
    duration_seconds INTEGER,
    is_360           BOOLEAN      DEFAULT FALSE,
    tags             TEXT[]       DEFAULT '{}',
    status           VARCHAR(20)  NOT NULL,  -- 'pending', 'processing', 'ready', 'error'
    published_at     TIMESTAMP WITH TIME ZONE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_contents_short_id   ON contents(short_id);
CREATE INDEX idx_contents_status     ON contents(status);
CREATE INDEX idx_contents_type       ON contents(content_type);
CREATE INDEX idx_contents_updated_at ON contents(updated_at);

-- =====================
-- 動画バリアント（ABR 各品質）
-- =====================
CREATE TABLE video_variants (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    content_id   UUID         NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    variant_type VARCHAR(20)  NOT NULL,     -- 'master'(マスタープレイリスト), 'original'(元品質), '1080p', '720p', '480p', 'audio'
    hls_key      TEXT         NOT NULL,     -- MinIO オブジェクトキー（.m3u8）
    bandwidth    INTEGER,                   -- bps（マスタープレイリスト用）
    resolution   VARCHAR(20),              -- '1920x1080' など（audio は NULL）
    codecs       VARCHAR(100),             -- 'avc1.640028,mp4a.40.2' など
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_video_variants_content_id ON video_variants(content_id);

-- =====================
-- アセット（サムネイル・ポスター等）
-- =====================
CREATE TABLE assets (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    content_id UUID        NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    asset_role VARCHAR(50) NOT NULL,  -- 'thumbnail', 'poster'
    s3_key     TEXT        NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_assets_content_id ON assets(content_id);

-- updated_at の自動更新トリガー
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER contents_updated_at
    BEFORE UPDATE ON contents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
