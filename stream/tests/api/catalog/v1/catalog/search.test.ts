import { describe, it, expect, beforeAll } from 'vitest';
import axios from 'axios';
import { loginAs, fullHeaders, cookieOnlyHeaders } from '../../../helpers/auth';
import type { AuthCredentials } from '../../../helpers/auth';

// GET /api/catalog/v1/catalog/search

const METADATA_URL = `${process.env.METADATA_API_URL || 'http://127.0.0.1:8000/api/metadata/v1'}/admin/metadata/contents`;
const CATALOG_URL = `${process.env.CATALOG_API_URL || 'http://127.0.0.1:8000/api/catalog/v1'}`;
const SEARCH_URL = `${CATALOG_URL}/catalog/search`;
const SYNC_URL = `${CATALOG_URL}/internal/catalog/sync`;

describe('GET /api/catalog/v1/catalog/search', () => {
  let creds: AuthCredentials;
  let shortId: string;
  const uniqueTitle = `検索テスト動画-${Date.now()}`;

  beforeAll(async () => {
    creds = await loginAs(`admin-catalog-search-test-${Date.now()}`);

    // コンテンツ作成
    const createRes = await axios.post(
      METADATA_URL,
      {
        content_type: 'video',
        title: uniqueTitle,
        description: '検索テスト用コンテンツ',
        video_details: { is_360: false, duration_seconds: 60, director: 'Test Director' },
      },
      { headers: fullHeaders(creds) },
    );
    shortId = createRes.data.short_id;

    // カタログへ同期
    await axios.post(`${SYNC_URL}/${shortId}`, null, { headers: fullHeaders(creds) });
  }, 30000);

  it('タイトルキーワードで検索すると同期済みコンテンツが見つかる', async () => {
    // 検索インデックスへの反映を待機（最大 20 秒）
    let found = false;
    for (let i = 0; i < 20; i++) {
      const res = await axios.get(`${SEARCH_URL}?q=${encodeURIComponent(uniqueTitle)}`, {
        headers: fullHeaders(creds),
        validateStatus: () => true,
      });
      if (res.status === 200 && Array.isArray(res.data)) {
        const match = res.data.find((c: any) => c.short_id === shortId);
        if (match) { found = true; break; }
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
