-- =====================
-- Baseline Schema
-- =====================

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'content_type') THEN
        CREATE TYPE content_type AS ENUM ('video', 'image_gallery', 'vr360', 'ebook');
    END IF;
END$$;

-- 全コンテンツ共通のベーステーブル (Class Table Inheritance: 親)
CREATE TABLE contents (
    id           UUID         PRIMARY KEY,
    short_id     VARCHAR(50)  UNIQUE NOT NULL,
    content_type content_type NOT NULL,
    title        VARCHAR(255) NOT NULL,
    description  TEXT,
    rating       DECIMAL(3,2),
    tags         TEXT[]       DEFAULT '{}',
    published_at TIMESTAMP WITH TIME ZONE,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_contents_short_id  ON contents(short_id);
CREATE INDEX idx_contents_updated_at ON contents(updated_at);

-- 動画固有データ (Class Table Inheritance: 子テーブル、content_id が PK かつ FK → 1:1)
CREATE TABLE content_videos (
    content_id       UUID         PRIMARY KEY REFERENCES contents(id) ON DELETE CASCADE,
    is_360           BOOLEAN      DEFAULT FALSE,
    duration_seconds INTEGER,
    director         VARCHAR(255)
);

-- 共通アセットテーブル (content_id FK → 1:N)
CREATE TABLE assets (
    id         UUID        PRIMARY KEY,
    content_id UUID        NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    asset_role VARCHAR(50) NOT NULL,
    s3_key     TEXT        NOT NULL,
    public_url TEXT        NOT NULL DEFAULT ''
);
CREATE INDEX idx_assets_content_id ON assets(content_id);
CREATE INDEX idx_assets_s3_key     ON assets(s3_key);

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

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_contents_updated_at
    BEFORE UPDATE ON contents
    FOR EACH ROW EXECUTE PROCEDURE set_updated_at();

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE PROCEDURE set_updated_at();

-- assets 更新時に親 contents の updated_at を更新する
CREATE OR REPLACE FUNCTION update_parent_content_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'DELETE') THEN
        UPDATE contents SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.content_id;
    ELSE
        UPDATE contents SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.content_id;
    END IF;
    RETURN NULL;
END;
$$ language 'plpgsql';

CREATE TRIGGER trigger_update_parent_content
    AFTER INSERT OR UPDATE OR DELETE ON assets
    FOR EACH ROW EXECUTE PROCEDURE update_parent_content_updated_at();

-- =====================
-- Sync View (Benthos ポーリング用)
-- =====================
-- Benthos が 10秒間隔でポーリングし、PostgreSQL → MongoDB / Elasticsearch へ同期するためのビュー
-- content_videos は LEFT JOIN (image_gallery など子テーブルを持たないコンテンツに対応)
CREATE OR REPLACE VIEW content_sync_view AS
SELECT
    c.id,
    c.short_id,
    c.content_type::text AS type,
    c.title,
    c.description,
    c.rating,
    c.updated_at,
    c.tags,
    cv.director,
    cv.is_360,
    cv.duration_seconds,
    (
        SELECT json_object_agg(asset_role, COALESCE(NULLIF(public_url, ''), s3_key))
        FROM assets
        WHERE content_id = c.id
    ) AS assets
FROM contents c
LEFT JOIN content_videos cv ON cv.content_id = c.id;
