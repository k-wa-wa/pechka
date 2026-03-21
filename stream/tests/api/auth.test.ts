import { describe, it, expect } from 'vitest';
import axios from 'axios';

const AUTH_API_URL = process.env.AUTH_API_URL || 'http://127.0.0.1:8000/api/v1/auth';
const DEV_PROXY_URL = process.env.DEV_PROXY_URL || 'http://127.0.0.1:8000';

describe('認証API E2Eテスト', () => {
  const testEmail = `test-${Date.now()}@example.com`;
  let accessToken: string;
  let cloudflareToken: string;

  it('Dev ProxyからCloudflare形式의JWTを取得できること', async () => {
    const res = await axios.post(`${DEV_PROXY_URL}/mock/token`, {
      email: testEmail
    });
    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty('token');
    cloudflareToken = res.data.token;
  });

  it('Cloudflare JWTをヘッダーにセットしてSessionを開始すると、App JWTが取得できること（JIT作成）', async () => {
    // 2. Set Cookie for Dev Proxy to handle Cloudflare Access simulation
    axios.defaults.headers.common['Cookie'] = `cf-access-token-mock=${cloudflareToken}`;

    // 3. Exchange for App JWT via auth-service
    const res = await axios.get(`${AUTH_API_URL}/session`);
    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty('access_token');
    accessToken = res.data.access_token;
  });

  it('取得したアクセストークンでmeエンドポイントにアクセスし現在のユーザー情報を取得できること', async () => {
    const res = await axios.get(`${AUTH_API_URL}/me`, {
      headers: {
        Authorization: `Bearer ${accessToken}`
      }
    });

    expect(res.status).toBe(200);
    expect(res.data.email).toBe(testEmail);
    expect(res.data).toHaveProperty('id');
  });

  it('有効なJWTなしでSessionを開始すると、ログインGUIへリダイレクトされること', async () => {
    const res = await axios.get(`${AUTH_API_URL}/session`, {
      headers: { 'Cookie': '' },
      maxRedirects: 0,
      validateStatus: (status) => status === 302,
    });
    expect(res.status).toBe(302);
    expect(res.headers.location).toContain('/dev-proxy/login');
  });

  it('不正なJWTでSessionを開始すると失敗すること', async () => {
    try {
      await axios.get(`${AUTH_API_URL}/session`, {
        headers: {
          'Cookie': `cf-access-token-mock=invalid-token`
        }
      });
      throw new Error('Should have failed');
    } catch (err: any) {
      expect(err.response.status).toBe(500);
    }
  });
});
