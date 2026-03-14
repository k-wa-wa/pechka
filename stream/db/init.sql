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
    published_at     TIMESTAMP WITH TIME ZONE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_videos_short_id ON videos(short_id);

CREATE TABLE galleries (
    id           UUID         PRIMARY KEY,
    short_id     VARCHAR(50)  UNIQUE NOT NULL,
    title        VARCHAR(255) NOT NULL,
    description  TEXT,
    rating       DECIMAL(3,2),
    published_at TIMESTAMP WITH TIME ZONE,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_galleries_short_id ON galleries(short_id);

CREATE TABLE video_assets (
    id          UUID         PRIMARY KEY,
    video_id    UUID         NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    asset_role  VARCHAR(50)  NOT NULL,
    s3_key      TEXT         NOT NULL,
    public_url  TEXT         NOT NULL DEFAULT ''
);
CREATE INDEX idx_video_assets_video_id ON video_assets(video_id);

CREATE TABLE gallery_assets (
    id          UUID         PRIMARY KEY,
    gallery_id  UUID         NOT NULL REFERENCES galleries(id) ON DELETE CASCADE,
    asset_role  VARCHAR(50)  NOT NULL,
    s3_key      TEXT         NOT NULL,
    public_url  TEXT         NOT NULL DEFAULT ''
);
CREATE INDEX idx_gallery_assets_gallery_id ON gallery_assets(gallery_id);

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
