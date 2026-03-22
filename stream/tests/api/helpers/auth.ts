import axios from 'axios';

const DEV_PROXY_URL = process.env.DEV_PROXY_URL || 'http://127.0.0.1:8000';
const AUTH_API_URL = process.env.AUTH_API_URL || 'http://127.0.0.1:8000/api/v1/auth';

export interface AuthCredentials {
  cfToken: string;
  appToken: string;
  cookieHeader: string;
}

/**
 * Dev Proxy 経由で CF トークンを取得し、App JWT と交換する。
 * email に '@' が含まれない場合は '@example.com' を補完する。
 */
export async function loginAs(username: string): Promise<AuthCredentials> {
  const email = username.includes('@') ? username : `${username}@example.com`;

  const mockRes = await axios.post(`${DEV_PROXY_URL}/mock/token`, { email });
  const cfToken: string = mockRes.data.token;
  const cookieHeader = `cf-access-token-mock=${cfToken}`;

  const sessionRes = await axios.get(`${AUTH_API_URL}/session`, {
    headers: { Cookie: cookieHeader },
  });
  const appToken: string = sessionRes.data.access_token;

  return { cfToken, appToken, cookieHeader };
}

/** CF cookie + Bearer token — for normal authenticated API requests */
export function fullHeaders(creds: AuthCredentials): Record<string, string> {
  return {
    Cookie: creds.cookieHeader,
    Authorization: `Bearer ${creds.appToken}`,
  };
}

/** CF cookie only (no Bearer) — use to test that the API itself returns 401 */
export function cookieOnlyHeaders(creds: AuthCredentials): Record<string, string> {
  return { Cookie: creds.cookieHeader };
}

/** Bearer token only */
export function bearerHeader(appToken: string): Record<string, string> {
  return { Authorization: `Bearer ${appToken}` };
}
