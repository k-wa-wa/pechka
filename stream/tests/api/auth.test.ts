import { describe, it, expect } from 'vitest';
import axios from 'axios';

const AUTH_API_URL = process.env.AUTH_API_URL || 'http://localhost:8000/api/v1/auth';

describe('認証API E2Eテスト', () => {
  const testEmail = `test-${Date.now()}@example.com`;
  const testPassword = 'password123';
  let accessToken: string;
  let refreshToken: string;

  it('新規ユーザー登録ができること', async () => {
    const payload = {
      email: testEmail,
      password: testPassword
    };

    const res = await axios.post(`${AUTH_API_URL}/register`, payload);
    expect(res.status).toBe(201);
    expect(res.data).toHaveProperty('access_token');
    expect(res.data).toHaveProperty('refresh_token');
  });

  it('登録したユーザーでログインができること', async () => {
    const payload = {
      email: testEmail,
      password: testPassword
    };

    const res = await axios.post(`${AUTH_API_URL}/login`, payload);
    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty('access_token');
    expect(res.data).toHaveProperty('refresh_token');

    accessToken = res.data.access_token;
    refreshToken = res.data.refresh_token;
  });

  it('meエンドポイントで現在のユーザー情報を取得できること', async () => {
    const res = await axios.get(`${AUTH_API_URL}/me`, {
      headers: {
        Authorization: `Bearer ${accessToken}`
      }
    });

    expect(res.status).toBe(200);
    expect(res.data.email).toBe(testEmail);
    expect(res.data).toHaveProperty('id');
    expect(res.data).toHaveProperty('role');
  });

  it('リフレッシュトークンで新しいアクセストークンを取得できること', async () => {
    const payload = {
      refresh_token: refreshToken
    };

    const res = await axios.post(`${AUTH_API_URL}/refresh`, payload);
    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty('access_token');
    expect(res.data).toHaveProperty('refresh_token');

    accessToken = res.data.access_token;
  });

  it('誤ったパスワードでログインが失敗すること', async () => {
    const payload = {
      email: testEmail,
      password: 'wrong-password'
    };

    try {
      await axios.post(`${AUTH_API_URL}/login`, payload);
      throw new Error('Should have failed');
    } catch (err: any) {
      expect(err.response.status).toBe(401);
    }
  });

  it('トークンなしでのmeエンドポイントアクセスが失敗すること', async () => {
    try {
      await axios.get(`${AUTH_API_URL}/me`);
      throw new Error('Should have failed');
    } catch (err: any) {
      expect(err.response.status).toBe(401);
    }
  });
});
