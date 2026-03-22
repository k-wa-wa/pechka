import { describe, it, expect, beforeAll } from 'vitest';
import axios from 'axios';
import { loginAs, fullHeaders, cookieOnlyHeaders } from '../../helpers/auth';
import type { AuthCredentials } from '../../helpers/auth';

// GET /api/v1/auth/me

const ME_URL = `${process.env.AUTH_API_URL || 'http://127.0.0.1:8000/api/v1/auth'}/me`;

describe('GET /api/v1/auth/me', () => {
  let creds: AuthCredentials;
  const email = `me-test-${Date.now()}@example.com`;

  beforeAll(async () => {
    creds = await loginAs(email);
  });

  it('Authorization ヘッダーなしで 401 が返る', async () => {
    // CF cookie あり (dev-proxy を通過) + Bearer なし → auth-service が 401 を返す
    const res = await axios.get(ME_URL, {
      headers: cookieOnlyHeaders(creds),
      validateStatus: () => true,
    });
    expect(res.status).toBe(401);
  });

  it.each([
    ['Token abc'],
    ['bearer abc'], // lowercase
    ['Bearertoken'],
  ])('不正な形式の Authorization ヘッダー "%s" で 401 が返る', async (header) => {
    const res = await axios.get(ME_URL, {
      headers: { ...cookieOnlyHeaders(creds), Authorization: header },
      validateStatus: () => true,
    });
    expect(res.status).toBe(401);
  });

  it('無効なトークンで 401 が返る', async () => {
    const res = await axios.get(ME_URL, {
      headers: { ...cookieOnlyHeaders(creds), Authorization: 'Bearer invalid.token.here' },
      validateStatus: () => true,
    });
    expect(res.status).toBe(401);
  });

  it('有効な App JWT で 200 とユーザー情報が返る', async () => {
    const res = await axios.get(ME_URL, { headers: fullHeaders(creds) });
    expect(res.status).toBe(200);
    expect(res.data.email).toBe(email);
    expect(res.data).toHaveProperty('id');
  });
});
