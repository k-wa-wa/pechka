import { describe, it, expect, beforeAll } from 'vitest';
import axios from 'axios';
import { loginAs, fullHeaders, cookieOnlyHeaders } from '../../../helpers/auth';
import type { AuthCredentials } from '../../../helpers/auth';

// GET /api/catalog/v1/catalog/search
// NFS インポート済みデータ (title_a, title_b, title_c, title_t00, title_t01) を使用

const CATALOG_URL = `${process.env.CATALOG_API_URL || 'http://127.0.0.1:8000/api/catalog/v1'}`;
const SEARCH_URL = `${CATALOG_URL}/catalog/search`;

describe('GET /api/catalog/v1/catalog/search', () => {
  let creds: AuthCredentials;

  beforeAll(async () => {
    // nfs-admin グループ所属ユーザーとしてログイン
    creds = await loginAs('nfs-editor');
  }, 30000);

  it('NFS インポート済みコンテンツのタイトルで検索できる', async () => {
    // title_a は NFS インポート時に必ず存在するタイトル
    let found = false;
    for (let i = 0; i < 20; i++) {
      const res = await axios.get(`${SEARCH_URL}?q=title_a`, {
        headers: fullHeaders(creds),
        validateStatus: () => true,
      });
      if (res.status === 200 && Array.isArray(res.data) && res.data.length > 0) {
        found = true;
        break;
      }
      await new Promise((r) => setTimeout(r, 1000));
    }
    expect(found).toBe(true);
  }, 25000);

  it('マッチしないキーワードでは空配列が返る', async () => {
    const res = await axios.get(`${SEARCH_URL}?q=XYZZY_NO_MATCH_EVER`, {
      headers: fullHeaders(creds),
    });
    expect(res.status).toBe(200);
    expect(Array.isArray(res.data)).toBe(true);
    expect(res.data.length).toBe(0);
  });

  it('Bearer トークンなしで 401 が返る', async () => {
    const res = await axios.get(`${SEARCH_URL}?q=test`, {
      headers: cookieOnlyHeaders(creds),
      validateStatus: () => true,
    });
    expect(res.status).toBe(401);
  });
});
