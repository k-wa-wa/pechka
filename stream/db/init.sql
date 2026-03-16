-- =====================
-- Baseline Schema
-- =====================

CREATE TABLE videos (
    id               UUID         PRIMARY KEY,
    short_id         VARCHAR(50)  UNIQUE NOT NULL,
    title            VARCHAR(255) NOT NULL,
    description      TEXT,
    rating           DECIMAL(3,2),
    is_360           BOOLEAN      DEFAULT FALSE,
    duration_seconds INTEGER,
    director         VARCHAR(255),
    tags             TEXT[]       DEFAULT '{}',
    published_at     TIMESTAMP WITH TIME ZONE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_videos_short_id ON videos(short_id);
CREATE INDEX idx_videos_updated_at ON videos(updated_at);

CREATE TABLE galleries (
    id           UUID         PRIMARY KEY,
    short_id     VARCHAR(50)  UNIQUE NOT NULL,
    title        VARCHAR(255) NOT NULL,
    description  TEXT,
    rating       DECIMAL(3,2),
    tags         TEXT[]       DEFAULT '{}',
    published_at TIMESTAMP WITH TIME ZONE,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_galleries_short_id ON galleries(short_id);
CREATE INDEX idx_galleries_updated_at ON galleries(updated_at);

CREATE TABLE video_assets (
    id          UUID         PRIMARY KEY,
    video_id    UUID         NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    asset_role  VARCHAR(50)  NOT NULL,
    s3_key      TEXT         NOT NULL,
    public_url  TEXT         NOT NULL DEFAULT ''
);
CREATE INDEX idx_video_assets_video_id ON video_assets(video_id);
CREATE INDEX idx_video_assets_s3_key ON video_assets(s3_key);

CREATE TABLE gallery_assets (
    id          UUID         PRIMARY KEY,
    gallery_id  UUID         NOT NULL REFERENCES galleries(id) ON DELETE CASCADE,
    asset_role  VARCHAR(50)  NOT NULL,
    s3_key      TEXT         NOT NULL,
    public_url  TEXT         NOT NULL DEFAULT ''
);
CREATE INDEX idx_gallery_assets_gallery_id ON gallery_assets(gallery_id);
CREATE INDEX idx_gallery_assets_s3_key ON gallery_assets(s3_key);

-- =====================
-- Auth Service Tables
-- =====================

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
        CREATE TYPE user_role AS ENUM ('user', 'admin');
    END IF;
END$$;

CREATE TABLE users (
    id            UUID         PRIMARY KEY,
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT         NOT NULL,
    role          user_role    NOT NULL DEFAULT 'user',
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_users_email ON users(email);

CREATE TABLE refresh_tokens (
    id         UUID         PRIMARY KEY,
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT         NOT NULL,
    expires_at TIMESTAMP    WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP    WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- =====================
-- Triggers for updated_at
-- =====================

-- Function to set updated_at
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for main tables
CREATE TRIGGER update_videos_updated_at BEFORE UPDATE ON videos FOR EACH ROW EXECUTE PROCEDURE set_updated_at();
CREATE TRIGGER update_galleries_updated_at BEFORE UPDATE ON galleries FOR EACH ROW EXECUTE PROCEDURE set_updated_at();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE PROCEDURE set_updated_at();

-- Functions to update parent updated_at
CREATE OR REPLACE FUNCTION update_parent_video_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'DELETE') THEN
        UPDATE videos SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.video_id;
    ELSE
        UPDATE videos SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.video_id;
    END IF;
    RETURN NULL;
END;
$$ language 'plpgsql';

CREATE OR REPLACE FUNCTION update_parent_gallery_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'DELETE') THEN
        UPDATE galleries SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.gallery_id;
    ELSE
        UPDATE galleries SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.gallery_id;
    END IF;
    RETURN NULL;
END;
$$ language 'plpgsql';

-- Triggers for asset tables
CREATE TRIGGER trigger_update_parent_video 
AFTER INSERT OR UPDATE OR DELETE ON video_assets 
FOR EACH ROW EXECUTE PROCEDURE update_parent_video_updated_at();

CREATE TRIGGER trigger_update_parent_gallery 
AFTER INSERT OR UPDATE OR DELETE ON gallery_assets 
FOR EACH ROW EXECUTE PROCEDURE update_parent_gallery_updated_at();

-- =====================
-- Sync View
-- =====================
CREATE OR REPLACE VIEW content_sync_view AS
SELECT 
    v.id, v.short_id, 'video' as type, v.title, v.description, v.rating, v.updated_at,
    v.director, v.is_360, v.duration_seconds, v.tags,
    (SELECT json_object_agg(asset_role, COALESCE(NULLIF(public_url, ''), s3_key)) FROM video_assets WHERE video_id = v.id) as assets
FROM videos v
UNION ALL
SELECT 
    g.id, g.short_id, 'gallery' as type, g.title, g.description, g.rating, g.updated_at,
    NULL as director, NULL as is_360, NULL as duration_seconds, g.tags,
    (SELECT json_object_agg(asset_role, COALESCE(NULLIF(public_url, ''), s3_key)) FROM gallery_assets WHERE gallery_id = g.id) as assets
FROM galleries g;
