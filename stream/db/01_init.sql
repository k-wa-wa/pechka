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
    visibility   VARCHAR(20)  DEFAULT 'public', -- public, group_only
    allowed_groups UUID[]     DEFAULT '{}',
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
-- Auth & RBAC Service Tables
-- =====================

-- ユーザーテーブル
CREATE TABLE users (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email        VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255),
    avatar_url   TEXT,
    status       VARCHAR(20)  NOT NULL DEFAULT 'active',
    last_login   TIMESTAMP WITH TIME ZONE,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_users_email ON users(email);

-- グループテーブル
CREATE TABLE groups (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ユーザーとグループの紐付け
CREATE TABLE user_groups (
    user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, group_id)
);

-- ロール定義
CREATE TABLE roles (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(50)  UNIQUE NOT NULL,
    description TEXT,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- グループとロールの紐付け
CREATE TABLE group_roles (
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    role_id  UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, role_id)
);

-- ユーザーとロールの直接の紐付け
CREATE TABLE user_roles (
    user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id  UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- 権限定義
CREATE TABLE permissions (
    id       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    resource VARCHAR(50) NOT NULL,
    action   VARCHAR(50) NOT NULL,
    UNIQUE (resource, action)
);

-- ロールと権限の紐付け
CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- シーディング (初期データ)
INSERT INTO groups (name, description) VALUES ('Administrators', 'System administrators group');
INSERT INTO roles (name, description) VALUES ('admin', 'Full access to all resources');
INSERT INTO roles (name, description) VALUES ('viewer', 'Read-only access to resources');

-- admin ロールに全ての権限を付与するための準備 (パーミッションは後で動的に追加される想定だが、ここではサンプル)
INSERT INTO permissions (resource, action) VALUES ('content', 'read'), ('content', 'write'), ('content', 'delete'), ('user', 'read'), ('user', 'write');
INSERT INTO role_permissions (role_id, permission_id) 
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'admin';

-- Administrators グループに admin ロールを付与
INSERT INTO group_roles (group_id, role_id)
SELECT g.id, r.id FROM groups g, roles r WHERE g.name = 'Administrators' AND r.name = 'admin';

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
    c.visibility,
    (
        SELECT array_agg(g.name)
        FROM groups g
        WHERE g.id = ANY(c.allowed_groups)
    ) AS allowed_groups,
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

-- =====================
-- Essential System Data
-- =====================

-- NFS Administrator group (required for NFS importer permissions)
INSERT INTO groups (id, name, description) 
VALUES ('550e8400-e29b-41d4-a716-446655441001', 'nfs-admin', 'NFS Video Administrators')
ON CONFLICT (name) DO NOTHING;
