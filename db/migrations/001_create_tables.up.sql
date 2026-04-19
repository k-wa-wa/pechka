-- Bluray disc management
CREATE TABLE discs (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    label      VARCHAR(255) NOT NULL UNIQUE,
    disc_name  VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Contents (each title/item)
CREATE TABLE contents (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    short_id         VARCHAR(50)  UNIQUE NOT NULL,
    content_type     VARCHAR(20)  NOT NULL DEFAULT 'video',
    -- 'video', 'image_gallery', 'vr360', 'document'
    disc_id          UUID         REFERENCES discs(id),
    title            VARCHAR(255) NOT NULL,
    description      TEXT         DEFAULT '',
    duration_seconds INTEGER,
    is_360           BOOLEAN      DEFAULT FALSE,
    tags             TEXT[]       DEFAULT '{}',
    status           VARCHAR(20)  NOT NULL DEFAULT 'pending',
    -- 'pending', 'processing', 'ready', 'error'
    published_at     TIMESTAMP WITH TIME ZONE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_contents_short_id   ON contents(short_id);
CREATE INDEX idx_contents_status     ON contents(status);
CREATE INDEX idx_contents_type       ON contents(content_type);
CREATE INDEX idx_contents_updated_at ON contents(updated_at);

-- Video variants (each ABR quality level)
CREATE TABLE video_variants (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    content_id   UUID         NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    variant_type VARCHAR(20)  NOT NULL,
    -- 'master', 'original', '1080p', '720p', '480p', 'audio'
    hls_key      TEXT         NOT NULL,
    bandwidth    INTEGER,
    resolution   VARCHAR(20),
    codecs       VARCHAR(100),
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_video_variants_content_id ON video_variants(content_id);

-- Assets (thumbnails, posters, etc.)
CREATE TABLE assets (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    content_id UUID        NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    asset_role VARCHAR(50) NOT NULL,
    -- 'thumbnail', 'poster'
    s3_key     TEXT        NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_assets_content_id ON assets(content_id);

-- Auto-update updated_at on contents changes
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
