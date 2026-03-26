import { describe, it, expect, beforeAll } from 'vitest';
import axios from 'axios';
import { loginAs, fullHeaders } from '../../../helpers/auth';
import type { AuthCredentials } from '../../../helpers/auth';

/**
 * カタログ可視性テスト
 *
 * NFS インポーターが取り込むコンテンツは visibility=group_only, allowed_groups=[nfs-admin UUID] で固定。
 * このテストは NFS インポート済みデータを使って RBAC フィルタリングを検証する。
 *
 * テストユーザー (02_seed_test.sql で定義):
 *   - nfs-editor : nfs-admin グループ所属 → グループ限定コンテンツ閲覧可
 *   - nfs-viewer : viewer ロールのみ、グループ非所属 → グループ限定コンテンツ閲覧不可
 */

// nfs-admin グループの固定 UUID
const NFS_ADMIN_GROUP_ID = '550e8400-e29b-41d4-a716-446655441001';

const CATALOG_URL = `${process.env.CATALOG_API_URL || 'http://127.0.0.1:8000/api/catalog/v1'}`;
const HOME_URL = `${CATALOG_URL}/catalog/home`;
const CONTENTS_URL = `${CATALOG_URL}/catalog/contents`;

describe('カタログ可視性: group_only コンテンツの RBAC フィルタリング', () => {
  let editorCreds: AuthCredentials;
  let viewerCreds: AuthCredentials;
  let nfsShortId: string;

  beforeAll(async () => {
    [editorCreds, viewerCreds] = await Promise.all([
      loginAs('nfs-editor'),
      loginAs('nfs-viewer'),
    ]);

    // nfs-editor としてホームを取得し、NFS インポート済み group_only コンテンツの short_id を確保
    const homeRes = await axios.get(HOME_URL, { headers: fullHeaders(editorCreds) });
    const items = homeRes.data.sections?.[0]?.items ?? [];
    const nfsContent = items.find((c: any) =>
      c.visibility === 'group_only' &&
      Array.isArray(c.allowed_groups) &&
      c.allowed_groups.includes(NFS_ADMIN_GROUP_ID)
    );
    if (!nfsContent) throw new Error('NFS インポート済みの group_only コンテンツが見つかりません');
    nfsShortId = nfsContent.short_id;
  }, 30000);

  it('nfs-editor (nfs-admin グループ所属) はカタログホームで NFS コンテンツを取得できる', async () => {
    const res = await axios.get(HOME_URL, { headers: fullHeaders(editorCreds) });
    expect(res.status).toBe(200);
    const items = res.data.sections?.[0]?.items ?? [];
    const found = items.some((c: any) => c.short_id === nfsShortId);
    expect(found).toBe(true);
  });

  it('nfs-viewer (グループ非所属) のホームには NFS コンテンツが含まれない', async () => {
    const res = await axios.get(HOME_URL, { headers: fullHeaders(viewerCreds) });
    expect(res.status).toBe(200);
    const items = res.data.sections?.[0]?.items ?? [];
    const found = items.some((c: any) => c.short_id === nfsShortId);
    expect(found).toBe(false);
  });

  it('nfs-editor は short_id で NFS コンテンツ詳細を取得できる', async () => {
    const res = await axios.get(`${CONTENTS_URL}/${nfsShortId}`, {
      headers: fullHeaders(editorCreds),
    });
    expect(res.status).toBe(200);
    expect(res.data.short_id).toBe(nfsShortId);
    expect(res.data.visibility).toBe('group_only');
    expect(res.data.allowed_groups).toContain(NFS_ADMIN_GROUP_ID);
  });

  it('nfs-viewer は group_only コンテンツ詳細を取得できない (404)', async () => {
    const res = await axios.get(`${CONTENTS_URL}/${nfsShortId}`, {
      headers: fullHeaders(viewerCreds),
      validateStatus: () => true,
    });
    expect(res.status).toBe(404);
  });
});
