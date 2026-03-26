-- =====================
-- Seed Test Data
-- =====================
-- このファイルはテスト用の初期データを投入する。
-- DB は make up 時に空の状態から起動するため、冪等処理は不要。

-- =====================
-- グループ
-- =====================

-- nfs-admin グループ (固定 UUID を持つ NFS 管理者グループ)
INSERT INTO groups (id, name, description)
VALUES ('550e8400-e29b-41d4-a716-446655441001', 'nfs-admin', 'NFS Video Administrators');

-- =====================
-- ロール
-- =====================

-- content-editor ロール: content:read + content:write + content:delete
INSERT INTO roles (name, description)
VALUES ('content-editor', 'Content creation and editing access');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'content-editor' AND p.resource = 'content';

-- nfs-admin グループに content-editor ロールを付与
INSERT INTO group_roles (group_id, role_id)
SELECT g.id, r.id FROM groups g, roles r
WHERE g.name = 'nfs-admin' AND r.name = 'content-editor';

-- =====================
-- テストユーザー
-- =====================
-- | ユーザー    | グループ        | ロール          | 権限                                    |
-- |------------|----------------|-----------------|----------------------------------------|
-- | sys-admin  | Administrators | admin (via grp) | 全権限                                  |
-- | nfs-editor | nfs-admin      | content-editor  | content:read, content:write, content:delete |
-- | nfs-viewer | -              | viewer          | content:read                           |
-- | outsider   | -              | -               | なし                                    |

INSERT INTO users (id, email, display_name, avatar_url, status) VALUES
    ('550e8400-e29b-41d4-a716-446655440010', 'sys-admin@example.com',  'System Administrator', '', 'active'),
    ('550e8400-e29b-41d4-a716-446655440011', 'nfs-editor@example.com', 'NFS Content Editor',   '', 'active'),
    ('550e8400-e29b-41d4-a716-446655440014', 'nfs-admin@example.com',  'NFS Admin',            '', 'active'),
    ('550e8400-e29b-41d4-a716-446655440012', 'nfs-viewer@example.com', 'NFS Viewer',           '', 'active'),
    ('550e8400-e29b-41d4-a716-446655440013', 'outsider@example.com',   'Outsider User',        '', 'active');

-- =====================
-- グループメンバーシップ
-- =====================

-- sys-admin → Administrators グループ
INSERT INTO user_groups (user_id, group_id)
SELECT '550e8400-e29b-41d4-a716-446655440010', id FROM groups WHERE name = 'Administrators';

-- nfs-editor → nfs-admin グループ
INSERT INTO user_groups (user_id, group_id)
VALUES ('550e8400-e29b-41d4-a716-446655440011', '550e8400-e29b-41d4-a716-446655441001');

-- nfs-admin → nfs-admin グループ
INSERT INTO user_groups (user_id, group_id)
VALUES ('550e8400-e29b-41d4-a716-446655440014', '550e8400-e29b-41d4-a716-446655441001');

-- =====================
-- ロール直接付与
-- =====================

-- nfs-viewer に viewer ロールを直接付与
INSERT INTO user_roles (user_id, role_id)
SELECT '550e8400-e29b-41d4-a716-446655440012', id FROM roles WHERE name = 'viewer';

-- outsider はロールなし
