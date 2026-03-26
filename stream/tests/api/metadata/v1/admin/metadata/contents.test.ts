import { describe, it, expect, beforeAll } from 'vitest';
import axios from 'axios';
import { loginAs, fullHeaders, cookieOnlyHeaders } from '../../../../helpers/auth';
import type { AuthCredentials } from '../../../../helpers/auth';

const METADATA_URL = process.env.METADATA_API_URL || 'http://127.0.0.1:8000/api/metadata/v1';
const CONTENTS_URL = `${METADATA_URL}/admin/metadata/contents`;

const NFS_ADMIN_GROUP_ID = '550e8400-e29b-41d4-a716-446655441001';

describe('GET /api/metadata/v1/admin/metadata/contents', () => {
  let adminCreds: AuthCredentials;
  let editorCreds: AuthCredentials;
  let viewerCreds: AuthCredentials;

  beforeAll(async () => {
    [adminCreds, editorCreds, viewerCreds] = await Promise.all([
      loginAs('sys-admin'),
      loginAs('nfs-editor'),
      loginAs('nfs-viewer'),
    ]);
  }, 30000);

  it('admin はコンテンツ一覧を取得できる', async () => {
    const res = await axios.get(CONTENTS_URL, { headers: fullHeaders(adminCreds) });
    expect(res.status).toBe(200);
    expect(Array.isArray(res.data)).toBe(true);
  });

  it('content:write 権限を持つ nfs-editor はコンテンツ一覧を取得できる', async () => {
    const res = await axios.get(CONTENTS_URL, { headers: fullHeaders(editorCreds) });
    expect(res.status).toBe(200);
    expect(Array.isArray(res.data)).toBe(true);
  });

  it('viewer は 403 を返す', async () => {
    const res = await axios.get(CONTENTS_URL, {
      headers: fullHeaders(viewerCreds),
      validateStatus: () => true,
    });
    expect(res.status).toBe(403);
  });

  it('Bearer トークンなしで 401 を返す', async () => {
    const res = await axios.get(CONTENTS_URL, {
      headers: cookieOnlyHeaders(adminCreds),
      validateStatus: () => true,
    });
    expect(res.status).toBe(401);
  });

  it('コンテンツには id / title / visibility / allowed_groups / group_permissions が含まれる', async () => {
    const res = await axios.get(CONTENTS_URL, { headers: fullHeaders(adminCreds) });
    expect(res.status).toBe(200);
    if (res.data.length > 0) {
      const c = res.data[0];
      expect(c).toHaveProperty('id');
      expect(c).toHaveProperty('title');
      expect(c).toHaveProperty('visibility');
      expect(c).toHaveProperty('allowed_groups');
      expect(c).toHaveProperty('group_permissions');
      expect(Array.isArray(c.allowed_groups)).toBe(true);
      expect(Array.isArray(c.group_permissions)).toBe(true);
    }
  });
});

describe('PUT /api/metadata/v1/admin/metadata/contents/:id', () => {
  let editorCreds: AuthCredentials;
  let viewerCreds: AuthCredentials;
  let contentId: string;

  beforeAll(async () => {
    [editorCreds, viewerCreds] = await Promise.all([
      loginAs('nfs-editor'),
      loginAs('nfs-viewer'),
    ]);

    const res = await axios.get(CONTENTS_URL, { headers: fullHeaders(editorCreds) });
    if (res.data.length === 0) throw new Error('テスト対象のコンテンツが存在しません');
    contentId = res.data[0].id;
  }, 30000);

  it('group_permissions に can_read=true を設定すると allowed_groups に反映される', async () => {
    const res = await axios.put(
      `${CONTENTS_URL}/${contentId}`,
      {
        title: 'test-update',
        description: '',
        rating: 0,
        tags: [],
        visibility: 'group_only',
        group_permissions: [
          { group_id: NFS_ADMIN_GROUP_ID, can_read: true, can_write: false, can_delete: false },
        ],
      },
      { headers: fullHeaders(editorCreds) },
    );
    expect(res.status).toBe(200);
    expect(res.data.visibility).toBe('group_only');
    expect(res.data.allowed_groups).toContain(NFS_ADMIN_GROUP_ID);
    expect(res.data.group_permissions).toHaveLength(1);
    expect(res.data.group_permissions[0].group_id).toBe(NFS_ADMIN_GROUP_ID);
    expect(res.data.group_permissions[0].can_read).toBe(true);
    expect(res.data.group_permissions[0].can_write).toBe(false);
  });

  it('can_read=false にすると allowed_groups から除外される', async () => {
    const res = await axios.put(
      `${CONTENTS_URL}/${contentId}`,
      {
        title: 'test-update',
        description: '',
        rating: 0,
        tags: [],
        visibility: 'group_only',
        group_permissions: [
          { group_id: NFS_ADMIN_GROUP_ID, can_read: false, can_write: true, can_delete: false },
        ],
      },
      { headers: fullHeaders(editorCreds) },
    );
    expect(res.status).toBe(200);
    expect(res.data.allowed_groups).not.toContain(NFS_ADMIN_GROUP_ID);
    expect(res.data.group_permissions[0].can_read).toBe(false);
    expect(res.data.group_permissions[0].can_write).toBe(true);
  });

  it('group_permissions を空にすると allowed_groups も空になる', async () => {
    const res = await axios.put(
      `${CONTENTS_URL}/${contentId}`,
      {
        title: 'test-update',
        description: '',
        rating: 0,
        tags: [],
        visibility: 'public',
        group_permissions: [],
      },
      { headers: fullHeaders(editorCreds) },
    );
    expect(res.status).toBe(200);
    expect(res.data.visibility).toBe('public');
    expect(res.data.allowed_groups).toHaveLength(0);
    expect(res.data.group_permissions).toHaveLength(0);
  });

  it('viewer は 403 を返す', async () => {
    const res = await axios.put(
      `${CONTENTS_URL}/${contentId}`,
      { title: 'x', description: '', rating: 0, tags: [], visibility: 'public', group_permissions: [] },
      { headers: fullHeaders(viewerCreds), validateStatus: () => true },
    );
    expect(res.status).toBe(403);
  });

  it('不正な UUID で 400 を返す', async () => {
    const res = await axios.put(
      `${CONTENTS_URL}/not-a-valid-uuid`,
      { title: 'x', description: '', rating: 0, tags: [], visibility: 'public', group_permissions: [] },
      { headers: fullHeaders(editorCreds), validateStatus: () => true },
    );
    expect(res.status).toBe(400);
  });
});
