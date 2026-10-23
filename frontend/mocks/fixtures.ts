import type {
  Channel,
  Content,
  MongoContent,
  MongoVariant,
  SearchResult,
  SubtitleCue,
  SubtitleTrack,
} from '@/lib/types'

// Shared sample data for Storybook stories and MSW handlers.
// Mirrors the shape served by e2e/mock-server.mjs (used for Playwright VRT)
// so that component behavior stays consistent across dev tools.

export const CHANNELS: Channel[] = [
  {
    id: 'tech-feed',
    name: 'Tech Feed',
    description: '自動収集・AI生成による技術ダイジェスト解説',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'bluray',
    name: 'Blu-ray',
    description: 'Blu-ray取り込み動画',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
]

export const CONTENTS: MongoContent[] = [
  {
    short_id: 'vid001',
    content_type: 'video',
    channel_id: 'bluray',
    channel_name: 'Blu-ray',
    title: '夏の思い出 2024',
    description: '海辺での夏のひとこまを収めた映像です。',
    duration_seconds: 342,
    is_360: false,
    tags: ['夏', '海', '思い出'],
    status: 'ready',
    disc_label: 'DISC-001',
    thumbnail_key: null,
    published_at: '2024-08-15T00:00:00Z',
    updated_at: '2024-08-20T10:00:00Z',
    has_subtitles: true,
  },
  {
    short_id: 'vid002',
    content_type: 'video',
    channel_id: 'bluray',
    channel_name: 'Blu-ray',
    title: '秋の紅葉ドライブ',
    description: '山道を走りながら撮影した紅葉の映像。',
    duration_seconds: 580,
    is_360: false,
    tags: ['秋', '紅葉', 'ドライブ'],
    status: 'ready',
    disc_label: null,
    thumbnail_key: null,
    published_at: '2024-11-03T00:00:00Z',
    updated_at: '2024-11-05T08:00:00Z',
    has_subtitles: false,
  },
  {
    short_id: 'gal001',
    content_type: 'image_gallery',
    channel_id: 'bluray',
    channel_name: 'Blu-ray',
    title: '春の花々フォトギャラリー',
    description: '桜・菜の花・チューリップなど春の花の写真集。',
    duration_seconds: null,
    is_360: false,
    tags: ['春', '花', '写真'],
    status: 'ready',
    disc_label: null,
    thumbnail_key: null,
    published_at: '2024-04-01T00:00:00Z',
    updated_at: '2024-04-02T09:00:00Z',
    has_subtitles: false,
  },
  {
    short_id: 'vr001',
    content_type: 'vr360',
    channel_id: 'bluray',
    channel_name: 'Blu-ray',
    title: '360° 富士山山頂の眺め',
    description: '富士山山頂からの360度パノラマ映像。',
    duration_seconds: 120,
    is_360: true,
    tags: ['富士山', 'VR', '360度'],
    status: 'ready',
    disc_label: null,
    thumbnail_key: null,
    published_at: '2024-07-20T00:00:00Z',
    updated_at: '2024-07-21T12:00:00Z',
    has_subtitles: false,
  },
  {
    short_id: 'vid003',
    content_type: 'video',
    channel_id: 'bluray',
    channel_name: 'Blu-ray',
    title: '冬の北海道旅行記',
    description: '雪景色の北海道を旅した記録映像。',
    duration_seconds: 1240,
    is_360: false,
    tags: ['冬', '北海道', '旅行'],
    status: 'ready',
    disc_label: 'DISC-002',
    thumbnail_key: null,
    published_at: '2024-01-10T00:00:00Z',
    updated_at: '2024-01-15T14:00:00Z',
    has_subtitles: false,
  },
  {
    short_id: 'doc001',
    content_type: 'document',
    channel_id: 'tech-feed',
    channel_name: 'Tech Feed',
    title: '2024年度 活動報告書',
    description: '年間の活動をまとめたドキュメント。',
    duration_seconds: null,
    is_360: false,
    tags: ['報告', 'ドキュメント'],
    status: 'ready',
    disc_label: null,
    thumbnail_key: null,
    document_key: 'resources/documents/doc001/document.md',
    document_mime_type: 'text/markdown',
    published_at: '2025-01-05T00:00:00Z',
    updated_at: '2025-01-06T09:00:00Z',
    has_subtitles: false,
  },
  {
    short_id: 'vid004',
    content_type: 'video',
    channel_id: 'bluray',
    channel_name: 'Blu-ray',
    title: '処理中のコンテンツ',
    description: '',
    duration_seconds: null,
    is_360: false,
    tags: [],
    status: 'processing',
    disc_label: null,
    thumbnail_key: null,
    published_at: null,
    updated_at: '2025-04-30T10:00:00Z',
    has_subtitles: false,
  },
]

export const ADMIN_CONTENTS: Content[] = CONTENTS.map((c, i) => ({
  id: `00000000-0000-0000-0000-${String(i + 1).padStart(12, '0')}`,
  short_id: c.short_id,
  content_type: c.content_type,
  channel_id: c.channel_id ?? 'bluray',
  disc_id: null,
  title: c.title,
  description: c.description,
  duration_seconds: c.duration_seconds,
  is_360: c.is_360,
  tags: c.tags,
  status: c.status,
  document_key: c.document_key ?? null,
  document_mime_type: c.document_mime_type ?? null,
  published_at: c.published_at,
  archived_at: null,
  created_at: c.updated_at,
  updated_at: c.updated_at,
}))

export const SAMPLE_MARKDOWN = `# 2024年度 活動報告書

本ドキュメントは **pechka** メディア基盤における年間の活動と技術構成をまとめたレポートです。

## 1. 主な成果

- **動画配信 (HLS / ABR)**: Blu-ray および自動生成ダイジェスト動画のマルチ品質配信
- **ドキュメント閲覧対応**: \`.md\` (Markdown) および \`.pdf\` (PDF) のブラウザ内表示に対応
- **全文検索**: Elasticsearch による高速キーワード・タグ検索

## 2. 対応フォーマット一覧

| フォーマット | 拡張子 | MIME タイプ | 状態 |
| --- | --- | --- | --- |
| Markdown | .md, .markdown | text/markdown | 対応済み |
| PDF | .pdf | application/pdf | 対応済み |

## 3. API からのアップロード例

\`\`\`bash
curl -X POST http://localhost:8080/api/v1/admin/contents/upload \\
  -H "X-Channel-Id: tech-feed" \\
  -F "file=@report.md" \\
  -F "title=2024年度 活動報告書" \\
  -F "tags=報告,ドキュメント"
\`\`\`

> 詳細は [pechka ドキュメント](https://example.com/docs) を参照してください。
`

export const VARIANTS: Record<string, MongoVariant[]> = {
  vid001: [
    { variant_type: 'master', hls_key: 'vid001/master.m3u8', bandwidth: null, resolution: null, codecs: null },
    { variant_type: '1080p', hls_key: 'vid001/1080p/index.m3u8', bandwidth: 5000000, resolution: '1920x1080', codecs: 'avc1.640028,mp4a.40.2' },
    { variant_type: '720p', hls_key: 'vid001/720p/index.m3u8', bandwidth: 2800000, resolution: '1280x720', codecs: 'avc1.4d401f,mp4a.40.2' },
    { variant_type: '480p', hls_key: 'vid001/480p/index.m3u8', bandwidth: 1400000, resolution: '854x480', codecs: 'avc1.4d401e,mp4a.40.2' },
  ],
  vid002: [
    { variant_type: 'master', hls_key: 'vid002/master.m3u8', bandwidth: null, resolution: null, codecs: null },
    { variant_type: '720p', hls_key: 'vid002/720p/index.m3u8', bandwidth: 2800000, resolution: '1280x720', codecs: 'avc1.4d401f,mp4a.40.2' },
  ],
  vr001: [
    { variant_type: 'master', hls_key: 'vr001/master.m3u8', bandwidth: null, resolution: null, codecs: null },
    { variant_type: '1080p', hls_key: 'vr001/1080p/index.m3u8', bandwidth: 5000000, resolution: '3840x1920', codecs: 'avc1.640028,mp4a.40.2' },
  ],
}

export const SEARCH_RESULTS: SearchResult[] = CONTENTS.map((c) => ({
  short_id: c.short_id,
  title: c.title,
  description: c.description,
  content_type: c.content_type,
  tags: c.tags,
  status: c.status,
}))

export const SUBTITLE_TRACKS: Record<string, SubtitleTrack[]> = {
  '00000000-0000-0000-0000-000000000001': [
    {
      id: 'track-vid001-ja',
      content_id: '00000000-0000-0000-0000-000000000001',
      language: 'ja',
      status: 'published',
      model: 'whisper-large-v3',
      created_at: '2024-08-20T10:00:00Z',
      updated_at: '2024-08-20T10:05:00Z',
    },
  ],
}

export const SUBTITLE_CUES: Record<string, SubtitleCue[]> = {
  'track-vid001-ja': [
    {
      id: 'cue-1',
      track_id: 'track-vid001-ja',
      seq: 1,
      start_ms: 0,
      end_ms: 3200,
      text: '夏の海辺にやってきました。',
      original_text: '夏の海辺にやってきました。',
      flagged: false,
      updated_at: '2024-08-20T10:05:00Z',
    },
    {
      id: 'cue-2',
      track_id: 'track-vid001-ja',
      seq: 2,
      start_ms: 3200,
      end_ms: 6800,
      text: '波の音がとても心地よいですね。',
      original_text: '波の音がとてもここちよいですね。',
      flagged: true,
      updated_at: '2024-08-20T10:05:00Z',
    },
    {
      id: 'cue-3',
      track_id: 'track-vid001-ja',
      seq: 3,
      start_ms: 6800,
      end_ms: 10500,
      text: '夕方まで散歩を楽しみました。',
      original_text: '夕方まで散歩を楽しみました。',
      flagged: false,
      updated_at: '2024-08-20T10:05:00Z',
    },
  ],
}
