import { test, expect } from '@playwright/test';
import axios from 'axios';
import { loginAs } from '../helpers/auth';

const BASE = process.env.BASE_URL || 'http://localhost:8000';
const METADATA_URL = `${BASE}/api/metadata/v1`;
const CATALOG_URL  = `${BASE}/api/catalog/v1`;
const NFS_ADMIN_GROUP_ID = '550e8400-e29b-41d4-a716-446655441001';

/**
 * カタログの visibility フィルタリング
 *
 * API 側の挙動（visibility=public/group_only）を
 * ブラウザコンテキストから間接的に確認する。
 * ※ カタログページが存在する場合は UI でも確認する。
 */

async function getHeaders(username: string) {
  const email = username.includes('@') ? username : `${username}@example.com`;
  const mockRes = await axios.post(`${BASE}/mock/token`, { email });
  const cfToken: string = mockRes.data.token;
  const cookie = `cf-access-token-mock=${cfToken}`;
  const sessionRes = await axios.get(`${BASE}/api/v1/auth/session`, {
    headers: { Cookie: cookie },
  });
  const appToken: string = sessionRes.data.access_token;
  return {
    Authorization: `Bearer ${appToken}`,
    Cookie: cookie,
  };
}

test.describe('Catalog visibility フィルタリング (API 経由)', () => {
  let publicShortId: string;
  let groupOnlyShortId: string;
  let groupOnlyId: string;

  test.beforeAll(async () => {
    const adminHeaders = await getHeaders('sys-admin');

    // public コンテンツ作成・同期
    const pubRes = await axios.post(
      `${METADATA_URL}/admin/metadata/contents`,
      {
        content_type: 'video',
        title: `E2E公開動画-${Date.now()}`,
        description: 'public',
        video_details: { is_360: false, duration_seconds: 10, director: 'E2E' },
      },
      { headers: adminHeaders },
    );
    publicShortId = pubRes.data.short_id;
    await axios.post(`${CATALOG_URL}/internal/catalog/sync/${publicShortId}`, null, { headers: adminHeaders });

    // group_only コンテンツ作成・更新・同期
    const grpRes = await axios.post(
      `${METADATA_URL}/admin/metadata/contents`,
      {
        content_type: 'video',
        title: `E2Eグループ限定動画-${Date.now()}`,
        description: 'group_only',
        video_details: { is_360: false, duration_seconds: 10, director: 'E2E' },
      },
      { headers: adminHeaders },
    );
    groupOnlyShortId = grpRes.data.short_id;
    groupOnlyId = grpRes.data.id;

    await axios.put(
      `${METADATA_URL}/admin/metadata/contents/${groupOnlyId}`,
      {
        content_type: 'video',
        title: `E2Eグループ限定動画`,
        description: 'group_only',
        visibility: 'group_only',
        allowed_groups: [NFS_ADMIN_GROUP_ID],
        video_details: { is_360: false, duration_seconds: 10, director: 'E2E' },
      },
      { headers: adminHeaders },
    );
    await axios.post(`${CATALOG_URL}/internal/catalog/sync/${groupOnlyShortId}`, null, { headers: adminHeaders });
  });

  test('nfs-editor は public コンテンツを取得できる', async ({ request }) => {
    const headers = await getHeaders('nfs-editor');
    const res = await request.get(`${CATALOG_URL}/catalog/contents/${publicShortId}`, { headers });
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.visibility).toBe('public');
  });

  test('nfs-viewer も public コンテンツを取得できる', async ({ request }) => {
    const headers = await getHeaders('nfs-viewer');
    const res = await request.get(`${CATALOG_URL}/catalog/contents/${publicShortId}`, { headers });
    expect(res.status()).toBe(200);
  });

  test('nfs-editor (nfs-admin グループ所属) は group_only コンテンツを取得できる', async ({ request }) => {
    const headers = await getHeaders('nfs-editor');
    const res = await request.get(`${CATALOG_URL}/catalog/contents/${groupOnlyShortId}`, { headers });
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.visibility).toBe('group_only');
  });

  test('nfs-viewer (nfs-admin グループ非所属) は group_only コンテンツを取得できない (404)', async ({ request }) => {
    const headers = await getHeaders('nfs-viewer');
    const res = await request.get(`${CATALOG_URL}/catalog/contents/${groupOnlyShortId}`, { headers });
    expect(res.status()).toBe(404);
  });

  test('visibility を public に変更・再同期すると nfs-viewer も取得できる', async ({ request }) => {
    const adminHeaders = await getHeaders('sys-admin');

    // public に変更
    await axios.put(
      `${METADATA_URL}/admin/metadata/contents/${groupOnlyId}`,
      {
        content_type: 'video',
        title: 'E2Eグループ限定動画',
        description: 'public に変更',
        visibility: 'public',
        allowed_groups: [],
        video_details: { is_360: false, duration_seconds: 10, director: 'E2E' },
      },
      { headers: adminHeaders },
    );
    await axios.post(`${CATALOG_URL}/internal/catalog/sync/${groupOnlyShortId}`, null, { headers: adminHeaders });

    const viewerHeaders = await getHeaders('nfs-viewer');
    const res = await request.get(`${CATALOG_URL}/catalog/contents/${groupOnlyShortId}`, { headers: viewerHeaders });
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.visibility).toBe('public');
  });
});
