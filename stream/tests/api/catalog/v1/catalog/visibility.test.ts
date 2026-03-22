import { describe, it, expect, beforeAll } from 'vitest';
import axios from 'axios';
import { loginAs, fullHeaders } from '../../../helpers/auth';
import type { AuthCredentials } from '../../../helpers/auth';

/**
 * Catalog Service: visibility フィルタリングのテスト
 *
 * シナリオ:
 *   1. public コンテンツ → 全ユーザーが閲覧できる
 *   2. group_only コンテンツ (nfs-admin グループ限定)
 *      → nfs-editor (nfs-admin グループ所属) は閲覧できる
 *      → nfs-viewer (nfs-admin グループ非所属) は閲覧できない (404)
 *   3. group_only → public に変更後、nfs-viewer も閲覧できる
 *
 * テストユーザー:
 *   - sys-admin  : admin ロール → コンテンツ作成・更新・同期に使用
 *   - nfs-editor : nfs-admin グループ所属 → グループ限定コンテンツ閲覧可
 *   - nfs-viewer : viewer ロールのみ、グループ非所属 → グループ限定コンテンツ閲覧不可
 */

const METADATA_BASE = process.env.METADATA_API_URL || 'http://127.0.0.1:8000/api/metadata/v1';
const CATALOG_BASE = process.env.CATALOG_API_URL || 'http://127.0.0.1:8000/api/catalog/v1';

const ADMIN_CONTENTS_URL = `${METADATA_BASE}/admin/metadata/contents`;
const CATALOG_CONTENTS_URL = `${CATALOG_BASE}/catalog/contents`;
const SYNC_URL = `${CATALOG_BASE}/internal/catalog/sync`;

// nfs-admin グループの固定 UUID
const NFS_ADMIN_GROUP_ID = '550e8400-e29b-41d4-a716-446655441001';

describe('Catalog: public コンテンツの可視性', () => {
  let adminCreds: AuthCredentials;
  let editorCreds: AuthCredentials;
  let viewerCreds: AuthCredentials;
  let publicShortId: string;

  beforeAll(async () => {
    [adminCreds, editorCreds, viewerCreds] = await Promise.all([
      loginAs('sys-admin'),
      loginAs('nfs-editor'),
      loginAs('nfs-viewer'),
    ]);

    // public コンテンツを作成・同期
    const createRes = await axios.post(
      ADMIN_CONTENTS_URL,
      {
        content_type: 'video',
        title: `公開動画テスト-${Date.now()}`,
        description: 'visibility=public のコンテンツ',
        video_details: { is_360: false, duration_seconds: 30, director: 'Test' },
      },
      { headers: fullHeaders(adminCreds) },
    );
    publicShortId = createRes.data.short_id;

    // visibility=public のまま同期
    await axios.post(`${SYNC_URL}/${publicShortId}`, null, {
      headers: fullHeaders(adminCreds),
    });
  }, 30000);

  it('nfs-editor (グループ所属) は public コンテンツを閲覧できる', async () => {
    const res = await axios.get(`${CATALOG_CONTENTS_URL}/${publicShortId}`, {
      headers: fullHeaders(editorCreds),
    });
    expect(res.status).toBe(200);
    expect(res.data.short_id).toBe(publicShortId);
    expect(res.data.visibility).toBe('public');
  });

  it('nfs-viewer (グループ非所属) も public コンテンツを閲覧できる', async () => {
    const res = await axios.get(`${CATALOG_CONTENTS_URL}/${publicShortId}`, {
      headers: fullHeaders(viewerCreds),
    });
    expect(res.status).toBe(200);
    expect(res.data.short_id).toBe(publicShortId);
  });
});

describe('Catalog: group_only コンテンツの可視性', () => {
  let adminCreds: AuthCredentials;
  let editorCreds: AuthCredentials;
  let viewerCreds: AuthCredentials;
  let restrictedShortId: string;
  let restrictedContentId: string;

  beforeAll(async () => {
    [adminCreds, editorCreds, viewerCreds] = await Promise.all([
      loginAs('sys-admin'),
      loginAs('nfs-editor'),
      loginAs('nfs-viewer'),
    ]);

    // group_only コンテンツを作成
    const createRes = await axios.post(
      ADMIN_CONTENTS_URL,
      {
        content_type: 'video',
        title: `グループ限定動画-${Date.now()}`,
        description: 'nfs-admin グループのみ閲覧可能',
        video_details: { is_360: false, duration_seconds: 45, director: 'Test' },
      },
      { headers: fullHeaders(adminCreds) },
    );
    restrictedContentId = createRes.data.id;
    restrictedShortId = createRes.data.short_id;

    // visibility=group_only, allowed_groups=[nfs-admin] に更新
    await axios.put(
      `${ADMIN_CONTENTS_URL}/${restrictedContentId}`,
      {
        content_type: 'video',
        title: `グループ限定動画-${Date.now()}`,
        description: 'nfs-admin グループのみ閲覧可能',
        visibility: 'group_only',
        allowed_groups: [NFS_ADMIN_GROUP_ID],
        video_details: { is_360: false, duration_seconds: 45, director: 'Test' },
      },
      { headers: fullHeaders(adminCreds) },
    );

    // カタログに同期 (visibility と allowed_groups も反映される)
    await axios.post(`${SYNC_URL}/${restrictedShortId}`, null, {
      headers: fullHeaders(adminCreds),
    });
  }, 30000);

  it('nfs-editor (nfs-admin グループ所属) はグループ限定コンテンツを閲覧できる', async () => {
    const res = await axios.get(`${CATALOG_CONTENTS_URL}/${restrictedShortId}`, {
      headers: fullHeaders(editorCreds),
    });
    expect(res.status).toBe(200);
    expect(res.data.short_id).toBe(restrictedShortId);
    expect(res.data.visibility).toBe('group_only');
    expect(res.data.allowed_groups).toContain('nfs-admin');
  });

  it('nfs-viewer (nfs-admin グループ非所属) はグループ限定コンテンツを閲覧できない (404)', async () => {
    const res = await axios.get(`${CATALOG_CONTENTS_URL}/${restrictedShortId}`, {
      headers: fullHeaders(viewerCreds),
      validateStatus: () => true,
    });
    expect(res.status).toBe(404);
  });

  it('sys-admin (admin ロール) はグループ限定コンテンツを管理 API で閲覧できる', async () => {
    // 管理 API は ACL フィルタリングなし (全コンテンツ取得)
    const res = await axios.get(`${ADMIN_CONTENTS_URL}/${restrictedShortId}`, {
      headers: fullHeaders(adminCreds),
    });
    expect(res.status).toBe(200);
    expect(res.data.visibility).toBe('group_only');
  });
});

describe('Catalog: visibility 変更後のフィルタリング更新', () => {
  let adminCreds: AuthCredentials;
  let viewerCreds: AuthCredentials;
  let shortId: string;
  let contentId: string;

  beforeAll(async () => {
    [adminCreds, viewerCreds] = await Promise.all([
      loginAs('sys-admin'),
      loginAs('nfs-viewer'),
    ]);

    // group_only コンテンツを作成・同期
    const createRes = await axios.post(
      ADMIN_CONTENTS_URL,
      {
        content_type: 'video',
        title: `可視性変更テスト-${Date.now()}`,
        description: '最初は group_only、後で public に変更',
        video_details: { is_360: false, duration_seconds: 20, director: 'Test' },
      },
      { headers: fullHeaders(adminCreds) },
    );
    contentId = createRes.data.id;
    shortId = createRes.data.short_id;

    await axios.put(
      `${ADMIN_CONTENTS_URL}/${contentId}`,
      {
        content_type: 'video',
        title: `可視性変更テスト`,
        description: '最初は group_only',
        visibility: 'group_only',
        allowed_groups: [NFS_ADMIN_GROUP_ID],
        video_details: { is_360: false, duration_seconds: 20, director: 'Test' },
      },
      { headers: fullHeaders(adminCreds) },
    );

    await axios.post(`${SYNC_URL}/${shortId}`, null, { headers: fullHeaders(adminCreds) });
  }, 30000);

  it('group_only 時点では nfs-viewer は閲覧できない', async () => {
    const res = await axios.get(`${CATALOG_CONTENTS_URL}/${shortId}`, {
      headers: fullHeaders(viewerCreds),
      validateStatus: () => true,
    });
    expect(res.status).toBe(404);
  });

  it('visibility を public に変更・再同期すると nfs-viewer も閲覧できる', async () => {
    // public に変更
    await axios.put(
      `${ADMIN_CONTENTS_URL}/${contentId}`,
      {
        content_type: 'video',
        title: '可視性変更テスト',
        description: 'public に変更',
        visibility: 'public',
        allowed_groups: [],
        video_details: { is_360: false, duration_seconds: 20, director: 'Test' },
      },
      { headers: fullHeaders(adminCreds) },
    );

    // 再同期
    await axios.post(`${SYNC_URL}/${shortId}`, null, { headers: fullHeaders(adminCreds) });

    // nfs-viewer が閲覧できるようになっているか確認
    const res = await axios.get(`${CATALOG_CONTENTS_URL}/${shortId}`, {
      headers: fullHeaders(viewerCreds),
    });
    expect(res.status).toBe(200);
    expect(res.data.short_id).toBe(shortId);
    expect(res.data.visibility).toBe('public');
  });
});
