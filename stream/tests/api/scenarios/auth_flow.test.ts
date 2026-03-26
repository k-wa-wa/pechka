import { describe, it, expect } from 'vitest';
import axios from 'axios';

/**
 * シナリオテスト: フル認証フロー
 *
 * 1. Dev Proxy でログイン → CF JWT 取得
 * 2. /api/v1/auth/session で App JWT 取得 (JIT プロビジョニング)
 * 3. /api/v1/auth/me でユーザー情報確認
 * 4. /api/catalog/v1/catalog/home でカタログ閲覧
 * 5. NFS インポート済みコンテンツを nfs-editor として取得
 * 6. /api/v1/auth/logout でセッション終了（リダイレクト確認）
 */

const DEV_PROXY_URL = process.env.DEV_PROXY_URL || 'http://127.0.0.1:8000';
const AUTH_URL = process.env.AUTH_API_URL || 'http://127.0.0.1:8000/api/v1/auth';
const CATALOG_URL = process.env.CATALOG_API_URL || 'http://127.0.0.1:8000/api/catalog/v1';

describe('シナリオ: ログイン → カタログ閲覧 → ログアウト', () => {
  const username = `admin-scenario-user-${Date.now()}`;
  const email = `${username}@example.com`;

  let cookieHeader: string;
  let appToken: string;

  it('Step 1: Dev Proxy でログインし CF JWT を取得できる', async () => {
    const res = await axios.post(`${DEV_PROXY_URL}/mock/token`, { email });
    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty('token');

    cookieHeader = `cf-access-token-mock=${res.data.token}`;
  });

  it('Step 2: CF JWT を使って App JWT を取得できる (JIT プロビジョニング)', async () => {
    const res = await axios.get(`${AUTH_URL}/session`, {
      headers: { Cookie: cookieHeader },
    });
    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty('access_token');

    appToken = res.data.access_token;
  });

  it('Step 3: App JWT で /me にアクセスしユーザー情報を確認できる', async () => {
    const res = await axios.get(`${AUTH_URL}/me`, {
      headers: {
        Cookie: cookieHeader,
        Authorization: `Bearer ${appToken}`,
      },
    });
    expect(res.status).toBe(200);
    expect(res.data.email).toBe(email);
    expect(res.data).toHaveProperty('id');
    expect(Array.isArray(res.data.roles)).toBe(true);
  });

  it('Step 4: カタログホームを閲覧できる', async () => {
    const res = await axios.get(`${CATALOG_URL}/catalog/home`, {
      headers: {
        Cookie: cookieHeader,
        Authorization: `Bearer ${appToken}`,
      },
    });
    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty('banners');
    expect(res.data).toHaveProperty('sections');
  });

  it('Step 5: nfs-editor として NFS インポート済みコンテンツをカタログから取得できる', async () => {
    // nfs-editor でログインして group_only コンテンツを取得
    const cfRes = await axios.post(`${DEV_PROXY_URL}/mock/token`, { email: 'nfs-editor@example.com' });
    const editorCookie = `cf-access-token-mock=${cfRes.data.token}`;
    const sessionRes = await axios.get(`${AUTH_URL}/session`, { headers: { Cookie: editorCookie } });
    const editorToken = sessionRes.data.access_token;

    const homeRes = await axios.get(`${CATALOG_URL}/catalog/home`, {
      headers: { Cookie: editorCookie, Authorization: `Bearer ${editorToken}` },
    });
    expect(homeRes.status).toBe(200);
    const items = homeRes.data.sections?.[0]?.items ?? [];
    const nfsContent = items.find((c: any) => c.visibility === 'group_only');
    expect(nfsContent).toBeDefined();

    // コンテンツ詳細を取得
    const detailRes = await axios.get(`${CATALOG_URL}/catalog/contents/${nfsContent.short_id}`, {
      headers: { Cookie: editorCookie, Authorization: `Bearer ${editorToken}` },
    });
    expect(detailRes.status).toBe(200);
    expect(detailRes.data.short_id).toBe(nfsContent.short_id);
  });

  it('Step 6: /api/v1/auth/logout でセッションを終了できる（302 で /cdn-cgi/access/logout へリダイレクト）', async () => {
    const res = await axios.get(`${AUTH_URL}/logout`, {
      headers: { Cookie: cookieHeader },
      maxRedirects: 0,
      validateStatus: (s) => s === 302,
    });
    expect(res.status).toBe(302);
    expect(res.headers.location).toContain('/cdn-cgi/access/logout');
  });
});
