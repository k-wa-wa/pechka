-- =====================
-- Seed Test Data
-- =====================

-- 既存のシードデータのクリーンアップ (必要に応じて)
DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE email IN ('admin@example.com', 'nfs-admin@example.com', 'user1@example.com', 'user2@example.com', 'user3@example.com'));
DELETE FROM user_groups WHERE user_id IN (SELECT id FROM users WHERE email IN ('admin@example.com', 'nfs-admin@example.com', 'user1@example.com', 'user2@example.com', 'user3@example.com'));
DELETE FROM users WHERE email IN ('admin@example.com', 'nfs-admin@example.com', 'user1@example.com', 'user2@example.com', 'user3@example.com');
DELETE FROM groups WHERE name = 'nfs-admin';

-- 1. ユーザーの作成
-- nfs-admin
INSERT INTO users (id, email, display_name, avatar_url, status) 
VALUES ('550e8400-e29b-41d4-a716-446655440001', 'nfs-admin@example.com', 'NFS Administrator', '', 'active');

-- Test Users
INSERT INTO users (id, email, display_name, avatar_url, status) VALUES 
('550e8400-e29b-41d4-a716-446655440002', 'user1@example.com', 'Test User 1', '', 'active'),
('550e8400-e29b-41d4-a716-446655440003', 'user2@example.com', 'Test User 2', '', 'active'),
('550e8400-e29b-41d4-a716-446655440004', 'user3@example.com', 'Test User 3', '', 'active');

-- (admin@example.com も必要なら再作成するが、権限は付与しない)
INSERT INTO users (id, email, display_name, avatar_url, status) 
VALUES ('550e8400-e29b-41d4-a716-446655440000', 'admin@example.com', 'Old Admin', '', 'active');

-- 2. グループの作成
INSERT INTO groups (id, name, description) 
VALUES ('550e8400-e29b-41d4-a716-446655441001', 'nfs-admin', 'NFS Video Administrators')
ON CONFLICT (name) DO NOTHING;

-- 3. ユーザーとグループの紐付け
-- nfs-admin を nfs-admin グループに追加
INSERT INTO user_groups (user_id, group_id) 
VALUES ('550e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655441001');

-- 4. ユーザーとロールの紐付け (任意)
-- nfs-admin に admin ロールを付与 (メタデータ操作等のため)
INSERT INTO user_roles (user_id, role_id)
SELECT '550e8400-e29b-41d4-a716-446655440001', id FROM roles WHERE name = 'admin';
