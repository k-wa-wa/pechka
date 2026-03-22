import { describe, it, expect, beforeAll } from 'vitest';
import axios from 'axios';
import { loginAs } from '../../helpers/auth';
import type { AuthCredentials } from '../../helpers/auth';

// GET /api/v1/auth/logout

const LOGOUT_URL = `${process.env.AUTH_API_URL || 'http://127.0.0.1:8000/api/v1/auth'}/logout`;

describe('GET /api/v1/auth/logout', () => {
  let creds: AuthCredentials;

  beforeAll(async () => {
    creds = await loginAs(`logout-test-${Date.now()}`);
  });

  it('302 で LOGOUT_REDIRECT_URL へリダイレクトされる', async () => {
    const res = await axios.get(LOGOUT_URL, {
      headers: { Cookie: creds.cookieHeader },
      maxRedirects: 0,
      validateStatus: (s) => s === 302,
    });
    expect(res.status).toBe(302);
    // LOGOUT_REDIRECT_URL = /cdn-cgi/access/logout (k8s configmap)
    expect(res.headers.location).toContain('/cdn-cgi/access/logout');
  });

  it('リダイレクト先の /cdn-cgi/access/logout がログインページへ遷移する', async () => {
    // logout → /cdn-cgi/access/logout (dev-proxy) → /dev-proxy/login
    const res = await axios.get(LOGOUT_URL, {
      headers: { Cookie: creds.cookieHeader },
      maxRedirects: 5,
      validateStatus: () => true,
    });
    // 最終的にログインページ (200) へ到達する
    expect(res.status).toBe(200);
  });
});
