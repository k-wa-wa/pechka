import { describe, it, expect, beforeAll } from 'vitest';
import axios from 'axios';
import { loginAs, fullHeaders, cookieOnlyHeaders } from '../../../helpers/auth';
import type { AuthCredentials } from '../../../helpers/auth';

// GET /api/catalog/v1/catalog/home

const CATALOG_HOME_URL = `${process.env.CATALOG_API_URL || 'http://127.0.0.1:8000/api/catalog/v1'}/catalog/home`;

describe('GET /api/catalog/v1/catalog/home', () => {
  let creds: AuthCredentials;

  beforeAll(async () => {
    creds = await loginAs(`catalog-home-test-${Date.now()}`);
  });

  it('banners と sections を含むレスポンスが返る', async () => {
    const res = await axios.get(CATALOG_HOME_URL, { headers: fullHeaders(creds) });
    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty('banners');
    expect(res.data).toHaveProperty('sections');
  });

  it('Bearer トークンなしで 401 が返る', async () => {
    // CF cookie あり (dev-proxy を通過) + Bearer なし → catalog-service が 401 を返す
    const res = await axios.get(CATALOG_HOME_URL, {
      headers: cookieOnlyHeaders(creds),
      validateStatus: () => true,
    });
    expect(res.status).toBe(401);
  });
});
