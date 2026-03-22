import { describe, it, expect, beforeAll } from 'vitest';
import axios from 'axios';
import { loginAs, fullHeaders, cookieOnlyHeaders } from '../../../helpers/auth';
import type { AuthCredentials } from '../../../helpers/auth';

// GET /api/catalog/v1/catalog/contents/:short_id

const METADATA_URL = `${process.env.METADATA_API_URL || 'http://127.0.0.1:8000/api/metadata/v1'}/admin/metadata/contents`;
const CATALOG_URL = `${process.env.CATALOG_API_URL || 'http://127.0.0.1:8000/api/catalog/v1'}`;
const CONTENTS_URL = `${CATALOG_URL}/catalog/contents`;
const SYNC_URL = `${CATALOG_URL}/internal/catalog/sync`;

describe('GET /api/catalog/v1/catalog/contents/:short_id', () => {
  let creds: AuthCredentials;
  let shortId: string;
  const title = `カタログ詳細テスト-${Date.now()}`;

  beforeAll(async () => {
    creds = await loginAs(`admin-catalog-contents-test-${Date.now()}`);

    // コンテンツ作成 & 同期
    const createRes = await axios.post(
      METADATA_URL,
      {
        content_type: 'video',
        title,
        description: 'カタログ詳細取得テスト用',
        video_details: { is_360: false, duration_seconds: 90, director: 'Test' },
      },
      { headers: fullHeaders(creds) },
    );
    shortId = createRes.data.short_id;

    await axios.post(`${SYNC_URL}/${shortId}`, null, { headers: fullHeaders(creds) });
  }, 30000);

  it('short_id でカタログ詳細を取得できる', async () => {
    const res = await axios.get(`${CONTENTS_URL}/${shortId}`, { headers: fullHeaders(creds) });
    expect(res.status).toBe(200);
    expect(res.data.short_id).toBe(shortId);
    expect(res.data.title).toBe(title);
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
