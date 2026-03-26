import { describe, it, expect, beforeAll } from 'vitest';
import axios from 'axios';
import { loginAs, fullHeaders, cookieOnlyHeaders } from '../../../helpers/auth';
import type { AuthCredentials } from '../../../helpers/auth';

// GET /api/catalog/v1/catalog/contents/:short_id
// NFS インポート済みデータ (title_a / title_b / title_c / title_t00 / title_t01 / test_master) を使用

const CATALOG_URL = `${process.env.CATALOG_API_URL || 'http://127.0.0.1:8000/api/catalog/v1'}`;
const CONTENTS_URL = `${CATALOG_URL}/catalog/contents`;

describe('GET /api/catalog/v1/catalog/contents/:short_id', () => {
  let creds: AuthCredentials;
  let shortId: string;

  beforeAll(async () => {
    // nfs-admin グループ所属ユーザーとしてログインし、NFS インポート済みコンテンツの short_id を取得
    creds = await loginAs('nfs-editor');

    const homeRes = await axios.get(`${CATALOG_URL}/catalog/home`, {
      headers: fullHeaders(creds),
    });
    const items = homeRes.data.sections?.[0]?.items ?? [];
    const nfsContent = items.find((c: any) => c.visibility === 'group_only');
    if (!nfsContent) throw new Error('NFS インポート済みの group_only コンテンツが見つかりません');
    shortId = nfsContent.short_id;
  }, 30000);

  it('short_id でカタログ詳細を取得できる', async () => {
    const res = await axios.get(`${CONTENTS_URL}/${shortId}`, { headers: fullHeaders(creds) });
    expect(res.status).toBe(200);
    expect(res.data.short_id).toBe(shortId);
  });

  it('存在しない short_id で 404 が返る', async () => {
    const res = await axios.get(`${CONTENTS_URL}/nonexistent-short-id`, {
      headers: fullHeaders(creds),
      validateStatus: () => true,
    });
    expect(res.status).toBe(404);
  });

  it('Bearer トークンなしで 401 が返る', async () => {
    const res = await axios.get(`${CONTENTS_URL}/${shortId}`, {
      headers: cookieOnlyHeaders(creds),
      validateStatus: () => true,
    });
    expect(res.status).toBe(401);
  });
});
