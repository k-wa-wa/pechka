import { describe, it, expect } from 'vitest';
import axios from 'axios';

const METADATA_API_URL = process.env.METADATA_API_URL || 'http://localhost:8000/api/metadata/v1';
const CATALOG_API_URL = process.env.CATALOG_API_URL || 'http://localhost:8000/api/catalog/v1';

describe('コンテンツAPI E2Eテスト', () => {
  let createdVideoId: string;
  let createdVideoShortId: string;

  it('新しい動画コンテンツを作成できること', async () => {
    const payload = {
      content_type: 'video',
      title: 'E2Eテスト用動画',
      description: 'Vitest E2Eで作成されました',
      video_details: {
        is_360: false,
        duration_seconds: 120,
        director: 'Vitest Runner'
      }
    };

    const res = await axios.post(`${METADATA_API_URL}/admin/metadata/contents`, payload);
    expect(res.status).toBe(201);
    expect(res.data).toHaveProperty('id');
    expect(res.data).toHaveProperty('short_id');
    expect(res.data.title).toBe('E2Eテスト用動画');
    expect(res.data.video_details.duration_seconds).toBe(120);

    createdVideoId = res.data.id;
    createdVideoShortId = res.data.short_id;
  });

  it('様々なコンテンツ種別を作成できること (Gallery, VR360, Ebook)', async () => {
    const types = ['image_gallery', 'vr360', 'ebook'];
    for (const type of types) {
      const payload: any = {
        content_type: type,
        title: `E2Eテスト用 ${type}`,
        description: `${type} の説明文`
      };
      if (type === 'vr360') {
        payload.video_details = {
          is_360: true,
          duration_seconds: 60,
          director: 'VR Dir'
        };
      }
      const res = await axios.post(`${METADATA_API_URL}/admin/metadata/contents`, payload);
      expect(res.status).toBe(201);
      expect(res.data.content_type).toBe(type);
    }
  });

  it('Metadata一覧から作成したコンテンツを取得できること', async () => {
    const res = await axios.get(`${METADATA_API_URL}/admin/metadata/contents`);
    expect(res.status).toBe(200);
    expect(Array.isArray(res.data)).toBe(true);

    const match = res.data.find((c: any) => c.id === createdVideoId);
    expect(match).toBeDefined();
    expect(match.title).toBe('E2Eテスト用動画');
  });

  it('short_id でコンテンツ詳細を取得できること', async () => {
    const res = await axios.get(`${METADATA_API_URL}/admin/metadata/contents/${createdVideoShortId}`);
    expect(res.status).toBe(200);
    expect(res.data.id).toBe(createdVideoId);
    expect(res.data.title).toBe('E2Eテスト用動画');
  });

  it('コンテンツ情報を更新できること', async () => {
    const updatePayload = {
      content_type: 'video',
      title: '更新後のタイトル',
      description: '更新後の説明文',
      video_details: {
        is_360: false,
        duration_seconds: 240,
        director: '新監督'
      }
    };
    const res = await axios.put(`${METADATA_API_URL}/admin/metadata/contents/${createdVideoId}`, updatePayload);
    expect(res.status).toBe(200);
    expect(res.data.title).toBe('更新後のタイトル');
    expect(res.data.video_details.duration_seconds).toBe(240);
  });

  it('コンテンツにアセットを追加できること', async () => {
    const payload = {
      assets: [
        {
          asset_role: 'hls_master',
          minio_key: 'videos/e2e/master.m3u8'
        }
      ]
    };

    const res = await axios.post(`${METADATA_API_URL}/admin/metadata/contents/${createdVideoId}/assets`, payload);
    expect(res.status).toBe(200);
    expect(res.data.message).toBe('assets added successfully');
  });

  it('手動同期をトリガーできること', async () => {
    const res = await axios.post(`${CATALOG_API_URL}/internal/catalog/sync/${createdVideoShortId}`);
    expect(res.status).toBe(200);
    expect(res.data.status).toBe('synchronized');
  });

  it('カタログホームを取得できること', async () => {
    const res = await axios.get(`${CATALOG_API_URL}/catalog/home`);
    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty('banners');
    expect(res.data).toHaveProperty('sections');
  });

  it('カタログ検索で同期されたデータが見つかること', async () => {
    let synced = false;
    for (let i = 0; i < 20; i++) {
      try {
        const res = await axios.get(`${CATALOG_API_URL}/catalog/search?q=更新後`);
        if (res.status === 200 && Array.isArray(res.data) && res.data.length > 0) {
          const match = res.data.find((c: any) => c.short_id === createdVideoShortId);
          if (match && match.title === '更新後のタイトル') {
            synced = true;
            break;
          }
        }
      } catch (err: any) {
        // ignore
      }
      await new Promise(resolve => setTimeout(resolve, 1000));
    }
    expect(synced).toBe(true);
  }, 25000);

  it('カタログ詳細を short_id で取得できること', async () => {
    const res = await axios.get(`${CATALOG_API_URL}/catalog/contents/${createdVideoShortId}`);
    expect(res.status).toBe(200);
    expect(res.data.short_id).toBe(createdVideoShortId);
    expect(res.data.title).toBe('更新後のタイトル');
  });
});
