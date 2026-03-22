import { describe, it, expect, beforeAll } from 'vitest';
import axios from 'axios';
import { loginAs, fullHeaders } from '../../../../helpers/auth';
import type { AuthCredentials } from '../../../../helpers/auth';

// /api/metadata/v1/admin/metadata/contents

const METADATA_URL = `${process.env.METADATA_API_URL || 'http://127.0.0.1:8000/api/metadata/v1'}/admin/metadata/contents`;

describe('/api/metadata/v1/admin/metadata/contents', () => {
  let creds: AuthCredentials;
  let createdId: string;
  let createdShortId: string;

  beforeAll(async () => {
    creds = await loginAs(`admin-metadata-test-${Date.now()}`);
  });

  describe('POST / - コンテンツ作成', () => {
    it('video コンテンツを作成できる', async () => {
      const res = await axios.post(
        METADATA_URL,
        {
          content_type: 'video',
          title: 'E2Eテスト用動画',
          description: 'Vitest E2Eで作成されました',
          video_details: {
            is_360: false,
            duration_seconds: 120,
            director: 'Vitest Runner',
          },
        },
        { headers: fullHeaders(creds) },
      );
      expect(res.status).toBe(201);
      expect(res.data).toHaveProperty('id');
      expect(res.data).toHaveProperty('short_id');
      expect(res.data.title).toBe('E2Eテスト用動画');
      expect(res.data.video_details.duration_seconds).toBe(120);

      createdId = res.data.id;
      createdShortId = res.data.short_id;
    });

    it.each(['image_gallery', 'vr360', 'ebook'])(
      'content_type=%s のコンテンツを作成できる',
      async (type) => {
        const payload: Record<string, unknown> = {
          content_type: type,
          title: `E2Eテスト用 ${type}`,
          description: `${type} の説明文`,
        };
        if (type === 'vr360') {
          payload.video_details = { is_360: true, duration_seconds: 60, director: 'VR Dir' };
        }
        const res = await axios.post(METADATA_URL, payload, { headers: fullHeaders(creds) });
        expect(res.status).toBe(201);
        expect(res.data.content_type).toBe(type);
      },
    );
  });

  describe('GET / - 一覧取得', () => {
    it('コンテンツ一覧に作成したコンテンツが含まれる', async () => {
      const res = await axios.get(METADATA_URL, { headers: fullHeaders(creds) });
      expect(res.status).toBe(200);
      expect(Array.isArray(res.data)).toBe(true);

      const match = res.data.find((c: any) => c.id === createdId);
      expect(match).toBeDefined();
      expect(match.title).toBe('E2Eテスト用動画');
    });
  });

  describe('GET /:short_id - 詳細取得', () => {
    it('short_id でコンテンツ詳細を取得できる', async () => {
      const res = await axios.get(`${METADATA_URL}/${createdShortId}`, {
        headers: fullHeaders(creds),
      });
      expect(res.status).toBe(200);
      expect(res.data.id).toBe(createdId);
      expect(res.data.title).toBe('E2Eテスト用動画');
    });
  });

  describe('PUT /:id - コンテンツ更新', () => {
    it('コンテンツ情報を更新できる', async () => {
      const res = await axios.put(
        `${METADATA_URL}/${createdId}`,
        {
          content_type: 'video',
          title: '更新後のタイトル',
          description: '更新後の説明文',
          video_details: {
            is_360: false,
            duration_seconds: 240,
            director: '新監督',
          },
        },
        { headers: fullHeaders(creds) },
      );
      expect(res.status).toBe(200);
      expect(res.data.title).toBe('更新後のタイトル');
      expect(res.data.video_details.duration_seconds).toBe(240);
    });
  });

  describe('POST /:id/assets - アセット追加', () => {
    it('コンテンツにアセットを追加できる', async () => {
      const res = await axios.post(
        `${METADATA_URL}/${createdId}/assets`,
        {
          assets: [{ asset_role: 'hls_master', minio_key: 'videos/e2e/master.m3u8' }],
        },
        { headers: fullHeaders(creds) },
      );
      expect(res.status).toBe(200);
      expect(res.data.message).toBe('assets added successfully');
    });
  });
});
