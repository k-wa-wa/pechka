import { test, expect } from '@playwright/test';
import axios from 'axios';
import { loginAs } from '../helpers/auth';

const BASE = process.env.BASE_URL || 'http://localhost:8000';
const CATALOG_URL  = `${BASE}/api/catalog/v1`;
const NFS_ADMIN_GROUP_ID = '550e8400-e29b-41d4-a716-446655441001';

/**
 * カタログの visibility フィルタリング
 *
 * NFS インポーター由来の group_only コンテンツを使って RBAC フィルタリングを検証する。
 * - nfs-editor : nfs-admin グループ所属 → グループ限定コンテンツ閲覧可
 * - nfs-viewer : グループ非所属 → グループ限定コンテンツ閲覧不可
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

test.describe('Catalog visibility フィルタリング (NFS データ使用)', () => {
  let groupOnlyShortId: string;

  test.beforeAll(async () => {
    // nfs-editor として NFS インポート済み group_only コンテンツの short_id を取得
    const editorHeaders = await getHeaders('nfs-editor');

    // リトライ: NFS インポーターが完了するまで待つ
    for (let i = 0; i < 20; i++) {
      const res = await axios.get(`${CATALOG_URL}/catalog/home`, {
        headers: editorHeaders,
        validateStatus: () => true,
      });
      const items = res.data?.sections?.[0]?.items ?? [];
      const nfsContent = items.find((c: any) =>
        c.visibility === 'group_only' &&
        Array.isArray(c.allowed_groups) &&
        c.allowed_groups.includes(NFS_ADMIN_GROUP_ID)
      );
      if (nfsContent) {
        groupOnlyShortId = nfsContent.short_id;
        break;
      }
      await new Promise((r) => setTimeout(r, 2000));
    }
    if (!groupOnlyShortId) throw new Error('NFS インポート済み group_only コンテンツが見つかりません');
  }, 60000);

  test('nfs-editor (nfs-admin グループ所属) は group_only コンテンツを取得できる', async ({ request }) => {
    const headers = await getHeaders('nfs-editor');
    const res = await request.get(`${CATALOG_URL}/catalog/contents/${groupOnlyShortId}`, { headers });
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.visibility).toBe('group_only');
    expect(body.allowed_groups).toContain(NFS_ADMIN_GROUP_ID);
  });

  test('nfs-viewer (nfs-admin グループ非所属) は group_only コンテンツを取得できない (404)', async ({ request }) => {
    const headers = await getHeaders('nfs-viewer');
    const res = await request.get(`${CATALOG_URL}/catalog/contents/${groupOnlyShortId}`, { headers });
    expect(res.status()).toBe(404);
  });

  test('nfs-viewer のホームには group_only コンテンツが含まれない', async ({ request }) => {
    const headers = await getHeaders('nfs-viewer');
    const res = await request.get(`${CATALOG_URL}/catalog/home`, { headers });
    expect(res.status()).toBe(200);
    const body = await res.json();
    const items: any[] = body.sections?.[0]?.items ?? [];
    expect(items.some((c: any) => c.short_id === groupOnlyShortId)).toBe(false);
  });

  test('nfs-editor のホームには group_only コンテンツが含まれる', async ({ request }) => {
    const headers = await getHeaders('nfs-editor');
    const res = await request.get(`${CATALOG_URL}/catalog/home`, { headers });
    expect(res.status()).toBe(200);
    const body = await res.json();
    const items: any[] = body.sections?.[0]?.items ?? [];
    expect(items.some((c: any) => c.short_id === groupOnlyShortId)).toBe(true);
  });
});
