import { describe, it, expect, beforeAll } from 'vitest';
import axios from 'axios';
import { loginAs, fullHeaders, cookieOnlyHeaders } from '../../../../helpers/auth';
import type { AuthCredentials } from '../../../../helpers/auth';

const METADATA_URL = process.env.METADATA_API_URL || 'http://127.0.0.1:8000/api/metadata/v1';
const GROUPS_URL = `${METADATA_URL}/admin/metadata/groups`;

describe('GET /api/metadata/v1/admin/metadata/groups', () => {
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

  it('admin はグループ一覧を取得できる', async () => {
    const res = await axios.get(GROUPS_URL, { headers: fullHeaders(adminCreds) });
    expect(res.status).toBe(200);
    expect(Array.isArray(res.data)).toBe(true);
    expect(res.data.length).toBeGreaterThan(0);
  });

  it('nfs-editor はグループ一覧を取得できる', async () => {
    const res = await axios.get(GROUPS_URL, { headers: fullHeaders(editorCreds) });
    expect(res.status).toBe(200);
    expect(Array.isArray(res.data)).toBe(true);
  });

  it('viewer は 403 を返す', async () => {
    const res = await axios.get(GROUPS_URL, {
      headers: fullHeaders(viewerCreds),
      validateStatus: () => true,
    });
    expect(res.status).toBe(403);
  });

  it('Bearer トークンなしで 401 を返す', async () => {
    const res = await axios.get(GROUPS_URL, {
      headers: cookieOnlyHeaders(adminCreds),
      validateStatus: () => true,
    });
    expect(res.status).toBe(401);
  });

  it('グループには id / name が含まれる', async () => {
    const res = await axios.get(GROUPS_URL, { headers: fullHeaders(adminCreds) });
    const g = res.data[0];
    expect(g).toHaveProperty('id');
    expect(g).toHaveProperty('name');
  });

  it('nfs-admin グループが含まれる', async () => {
    const res = await axios.get(GROUPS_URL, { headers: fullHeaders(adminCreds) });
    const names: string[] = res.data.map((g: any) => g.name);
    expect(names).toContain('nfs-admin');
  });
});
