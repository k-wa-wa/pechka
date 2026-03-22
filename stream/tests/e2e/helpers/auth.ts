import axios from 'axios';
import type { BrowserContext, Page } from '@playwright/test';

const BASE = process.env.BASE_URL || 'http://localhost:8000';

export interface E2ECredentials {
  cfToken: string;
  appToken: string;
}

/**
 * dev-proxy 経由でCFトークンとApp JWTを取得する。
 */
export async function getCredentials(username: string): Promise<E2ECredentials> {
  const email = username.includes('@') ? username : `${username}@example.com`;
  const mockRes = await axios.post(`${BASE}/mock/token`, { email });
  const cfToken: string = mockRes.data.token;

  const sessionRes = await axios.get(`${BASE}/api/v1/auth/session`, {
    headers: { Cookie: `cf-access-token-mock=${cfToken}` },
  });
  const appToken: string = sessionRes.data.access_token;

  return { cfToken, appToken };
}

/**
 * BrowserContext にCFクッキーをセットし、
 * 各 Page に localStorage の app_jwt を事前注入するスクリプトを登録する。
 *
 * addInitScript はページナビゲーション前に実行されるため、
 * admin ページが mount 時に fetchContents() を呼ぶ際に JWT が使える。
 */
export async function loginAs(
  ctx: BrowserContext,
  username: string,
): Promise<E2ECredentials> {
  const creds = await getCredentials(username);

  // CF cookie をブラウザにセット
  await ctx.addCookies([
    {
      name: 'cf-access-token-mock',
      value: creds.cfToken,
      domain: 'localhost',
      path: '/',
    },
  ]);

  // 全ページの localStorage に app_jwt を注入（ページロード前に実行される）
  await ctx.addInitScript((token) => {
    localStorage.setItem('app_jwt', token);
  }, creds.appToken);

  return creds;
}

/** ユーザー情報が Navbar に表示されるまで待つ（auth フロー完了の目印） */
export async function waitForAuth(page: Page): Promise<void> {
  // AuthProvider が user をセットすると Navbar にユーザー名が表示される
  // AuthProvider が user をセットするとプロフィールボタンが表示される
  await page.waitForSelector('nav button.bg-surface', { timeout: 15000 });
}
