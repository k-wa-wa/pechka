import { describe, it, expect } from 'vitest';
import axios from 'axios';

// GET /api/v1/auth/session

const DEV_PROXY_URL = process.env.DEV_PROXY_URL || 'http://127.0.0.1:8000';
const SESSION_URL = `${process.env.AUTH_API_URL || 'http://127.0.0.1:8000/api/v1/auth'}/session`;

describe('GET /api/v1/auth/session', () => {
  it('CF JWT Cookie が未セットの場合、ログインページへリダイレクト (302) される', async () => {
    const res = await axios.get(SESSION_URL, {
      headers: { Cookie: '' },
      maxRedirects: 0,
      validateStatus: (s) => s === 302,
    });
    expect(res.status).toBe(302);
    expect(res.headers.location).toContain('/dev-proxy/login');
  });

  it('不正な CF JWT を渡すと 500 が返る', async () => {
    const res = await axios.get(SESSION_URL, {
      headers: { Cookie: 'cf-access-token-mock=invalid-token' },
      validateStatus: () => true,
    });
    expect(res.status).toBe(500);
  });

  it('有効な CF JWT を渡すと 200 と App JWT が返る', async () => {
    const mockRes = await axios.post(`${DEV_PROXY_URL}/mock/token`, {
      email: `session-test-${Date.now()}@example.com`,
    });
    const cfToken: string = mockRes.data.token;

    const res = await axios.get(SESSION_URL, {
      headers: { Cookie: `cf-access-token-mock=${cfToken}` },
    });
    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty('access_token');
    expect(typeof res.data.access_token).toBe('string');
  });
});
