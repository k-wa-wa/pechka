import { describe, it, expect, beforeAll } from 'vitest';
import axios from 'axios';
import { loginAs, fullHeaders, cookieOnlyHeaders } from '../../../../helpers/auth';
import type { AuthCredentials } from '../../../../helpers/auth';

/**
 * 権限 (visibility / allowed_groups) 管理 API のテスト
 *
 * テストユーザー (02_seed_test.sql で定義):
 *   - sys-admin  : Administrators グループ → admin ロール → 全権限
 *   - nfs-editor : nfs-admin グループ → content-editor ロール → content:read + content:write
 *   - nfs-viewer : viewer ロール直接付与 → content:read のみ (管理画面アクセス不可)
 *   - outsider   : ロール・グループなし (管理画面アクセス不可)
 */

const METADATA_BASE = process.env.METADATA_API_URL || 'http://127.0.0.1:8000/api/metadata/v1';
const CONTENTS_URL = `${METADATA_BASE}/admin/metadata/contents`;
const GROUPS_URL = `${METADATA_BASE}/admin/metadata/groups`;

// nfs-admin グループの固定 UUID (02_seed_test.sql / 01_init.sql で定義)
const NFS_ADMIN_GROUP_ID = '550e8400-e29b-41d4-a716-446655441001';

describe('Admin Metadata: アクセス制御', () => {
  let adminCreds: AuthCredentials;
  let editorCreds: AuthCredentials;
  let viewerCreds: AuthCredentials;
  let outsiderCreds: AuthCredentials;

  beforeAll(async () => {
    [adminCreds, editorCreds, viewerCreds, outsiderCreds] = await Promise.all([
      loginAs('sys-admin'),
      loginAs('nfs-editor'),
      loginAs('nfs-viewer'),
      loginAs('outsider'),
    ]);
  }, 30000);

  describe('GET /admin/metadata/groups - グループ一覧', () => {
    it('sys-admin はグループ一覧を取得できる', async () => {
      const res = await axios.get(GROUPS_URL, { headers: fullHeaders(adminCreds) });
      expect(res.status).toBe(200);
      expect(Array.isArray(res.data)).toBe(true);
      expect(res.data.length).toBeGreaterThan(0);

      const nfsAdmin = res.data.find((g: any) => g.name === 'nfs-admin');
      expect(nfsAdmin).toBeDefined();
      expect(nfsAdmin.id).toBe(NFS_ADMIN_GROUP_ID);
    });

    it('nfs-editor (content:write) はグループ一覧を取得できる', async () => {
      const res = await axios.get(GROUPS_URL, { headers: fullHeaders(editorCreds) });
      expect(res.status).toBe(200);
      expect(Array.isArray(res.data)).toBe(true);
    });

    it('nfs-viewer (content:read のみ) は 403 を受け取る', async () => {
      const res = await axios.get(GROUPS_URL, {
        headers: fullHeaders(viewerCreds),
        validateStatus: () => true,
      });
      expect(res.status).toBe(403);
    });

    it('outsider (権限なし) は 403 を受け取る', async () => {
      const res = await axios.get(GROUPS_URL, {
        headers: fullHeaders(outsiderCreds),
        validateStatus: () => true,
      });
      expect(res.status).toBe(403);
    });

    it('CF JWT はあるが Bearer トークンなしで 401 を受け取る', async () => {
      // dev-proxy は CF JWT cookie で通過するが、API 側で Bearer がなければ 401
      const res = await axios.get(GROUPS_URL, {
        headers: cookieOnlyHeaders(adminCreds),
        validateStatus: () => true,
      });
      expect(res.status).toBe(401);
    });
  });

  describe('GET /admin/metadata/contents - 管理コンテンツ一覧 (アクセス制御)', () => {
    it('sys-admin はコンテンツ一覧にアクセスできる', async () => {
      const res = await axios.get(CONTENTS_URL, { headers: fullHeaders(adminCreds) });
      expect(res.status).toBe(200);
      expect(Array.isArray(res.data)).toBe(true);
    });

    it('nfs-editor (content:write) はコンテンツ一覧にアクセスできる', async () => {
      const res = await axios.get(CONTENTS_URL, { headers: fullHeaders(editorCreds) });
      expect(res.status).toBe(200);
      expect(Array.isArray(res.data)).toBe(true);
    });

    it('nfs-viewer (content:read のみ) は 403 を受け取る', async () => {
      const res = await axios.get(CONTENTS_URL, {
        headers: fullHeaders(viewerCreds),
        validateStatus: () => true,
      });
      expect(res.status).toBe(403);
    });

    it('outsider (権限なし) は 403 を受け取る', async () => {
      const res = await axios.get(CONTENTS_URL, {
        headers: fullHeaders(outsiderCreds),
        validateStatus: () => true,
      });
      expect(res.status).toBe(403);
    });
  });
});

describe('Admin Metadata: visibility / allowed_groups の CRUD', () => {
  let creds: AuthCredentials;
  let contentId: string;
  let contentShortId: string;

  beforeAll(async () => {
    creds = await loginAs('sys-admin');

    // テスト用コンテンツを作成
    const res = await axios.post(
      CONTENTS_URL,
      {
        content_type: 'video',
        title: `権限テスト用動画-${Date.now()}`,
        description: 'visibility テスト用コンテンツ',
        video_details: { is_360: false, duration_seconds: 60, director: 'Test' },
      },
      { headers: fullHeaders(creds) },
    );
    contentId = res.data.id;
    contentShortId = res.data.short_id;
  }, 30000);

  it('コンテンツはデフォルトで visibility=public として作成される', async () => {
    const res = await axios.get(`${CONTENTS_URL}/${contentShortId}`, {
      headers: fullHeaders(creds),
    });
    expect(res.status).toBe(200);
    expect(res.data.visibility).toBe('public');
    expect(res.data.allowed_groups).toEqual([]);
  });

  it('visibility=group_only + allowed_groups を更新できる', async () => {
    const res = await axios.put(
      `${CONTENTS_URL}/${contentId}`,
      {
        content_type: 'video',
        title: '権限更新テスト',
        description: 'group_only に更新',
        visibility: 'group_only',
        allowed_groups: [NFS_ADMIN_GROUP_ID],
        video_details: { is_360: false, duration_seconds: 60, director: 'Test' },
      },
      { headers: fullHeaders(creds) },
    );
    expect(res.status).toBe(200);
    expect(res.data.visibility).toBe('group_only');
    expect(res.data.allowed_groups).toContain(NFS_ADMIN_GROUP_ID);
  });

  it('更新後に GET で visibility と allowed_groups が正しく返る', async () => {
    const res = await axios.get(`${CONTENTS_URL}/${contentShortId}`, {
      headers: fullHeaders(creds),
    });
    expect(res.status).toBe(200);
    expect(res.data.visibility).toBe('group_only');
    expect(res.data.allowed_groups).toContain(NFS_ADMIN_GROUP_ID);
  });

  it('visibility=public に戻すと allowed_groups が空になる', async () => {
    const res = await axios.put(
      `${CONTENTS_URL}/${contentId}`,
      {
        content_type: 'video',
        title: '権限更新テスト',
        description: 'public に戻す',
        visibility: 'public',
        allowed_groups: [],
        video_details: { is_360: false, duration_seconds: 60, director: 'Test' },
      },
      { headers: fullHeaders(creds) },
    );
    expect(res.status).toBe(200);
    expect(res.data.visibility).toBe('public');
    expect(res.data.allowed_groups).toEqual([]);
  });

  it('nfs-editor (content:write) も visibility を更新できる', async () => {
    const editorCreds = await loginAs('nfs-editor');
    const res = await axios.put(
      `${CONTENTS_URL}/${contentId}`,
      {
        content_type: 'video',
        title: 'editor による更新',
        description: 'nfs-editor による visibility 更新',
        visibility: 'group_only',
        allowed_groups: [NFS_ADMIN_GROUP_ID],
        video_details: { is_360: false, duration_seconds: 60, director: 'Test' },
      },
      { headers: fullHeaders(editorCreds) },
    );
    expect(res.status).toBe(200);
    expect(res.data.visibility).toBe('group_only');
  });

  it('nfs-viewer (content:read のみ) は PUT で 403 を受け取る', async () => {
    const viewerCreds = await loginAs('nfs-viewer');
    const res = await axios.put(
      `${CONTENTS_URL}/${contentId}`,
      {
        content_type: 'video',
        title: '不正更新',
        description: '権限なし',
        visibility: 'public',
        allowed_groups: [],
      },
      {
        headers: fullHeaders(viewerCreds),
        validateStatus: () => true,
      },
    );
    expect(res.status).toBe(403);
  });

  it('visibility 未指定の場合は public として扱われる', async () => {
    // visibility フィールドなしで PUT
    const res = await axios.put(
      `${CONTENTS_URL}/${contentId}`,
      {
        content_type: 'video',
        title: 'visibility未指定',
        description: 'omit visibility field',
      },
      { headers: fullHeaders(creds) },
    );
    expect(res.status).toBe(200);
    expect(res.data.visibility).toBe('public');
  });
});
