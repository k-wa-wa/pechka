import http from 'node:http'

const CONTENTS = [
  {
    short_id: 'vid001',
    content_type: 'video',
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
  },
  {
    short_id: 'vid002',
    content_type: 'video',
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
  },
  {
    short_id: 'gal001',
    content_type: 'image_gallery',
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
  },
  {
    short_id: 'vr001',
    content_type: 'vr360',
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
  },
  {
    short_id: 'vid003',
    content_type: 'video',
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
  },
  {
    short_id: 'doc001',
    content_type: 'document',
    title: '2024年度 活動報告書',
    description: '年間の活動をまとめたドキュメント。',
    duration_seconds: null,
    is_360: false,
    tags: ['報告', 'ドキュメント'],
    status: 'ready',
    disc_label: null,
    thumbnail_key: null,
    published_at: '2025-01-05T00:00:00Z',
    updated_at: '2025-01-06T09:00:00Z',
  },
  {
    short_id: 'vid004',
    content_type: 'video',
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
  },
]

const ADMIN_CONTENTS = CONTENTS.map((c, i) => ({
  id: `00000000-0000-0000-0000-${String(i + 1).padStart(12, '0')}`,
  short_id: c.short_id,
  content_type: c.content_type,
  disc_id: null,
  title: c.title,
  description: c.description,
  duration_seconds: c.duration_seconds,
  is_360: c.is_360,
  tags: c.tags,
  status: c.status,
  published_at: c.published_at,
  created_at: c.updated_at,
  updated_at: c.updated_at,
}))

const VARIANTS = {
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

function send(res, status, data) {
  res.writeHead(status, { 'Content-Type': 'application/json' })
  res.end(JSON.stringify(data))
}

const server = http.createServer((req, res) => {
  const url = new URL(req.url, 'http://localhost')
  const path = url.pathname

  // GET /api/v1/contents
  if (path === '/api/v1/contents') {
    const contentType = url.searchParams.get('content_type')
    const limit = parseInt(url.searchParams.get('limit') ?? '100')
    let items = CONTENTS
    if (contentType) items = items.filter((c) => c.content_type === contentType)
    send(res, 200, items.slice(0, limit))
    return
  }

  // GET /api/v1/contents/:shortId
  const contentMatch = path.match(/^\/api\/v1\/contents\/([^/]+)$/)
  if (contentMatch) {
    const item = CONTENTS.find((c) => c.short_id === contentMatch[1])
    if (!item) { send(res, 404, { error: 'not found' }); return }
    send(res, 200, item)
    return
  }

  // GET /api/v1/contents/:shortId/variants
  const variantsMatch = path.match(/^\/api\/v1\/contents\/([^/]+)\/variants$/)
  if (variantsMatch) {
    const variants = VARIANTS[variantsMatch[1]] ?? []
    send(res, 200, variants)
    return
  }

  // GET /api/v1/admin/contents
  if (path === '/api/v1/admin/contents') {
    const limit = parseInt(url.searchParams.get('limit') ?? '200')
    send(res, 200, ADMIN_CONTENTS.slice(0, limit))
    return
  }

  // GET /api/v1/search
  if (path === '/api/v1/search') {
    const q = (url.searchParams.get('q') ?? '').toLowerCase()
    const results = CONTENTS.filter(
      (c) =>
        c.title.toLowerCase().includes(q) ||
        c.description.toLowerCase().includes(q) ||
        c.tags.some((t) => t.toLowerCase().includes(q))
    ).map((c) => ({
      short_id: c.short_id,
      title: c.title,
      description: c.description,
      content_type: c.content_type,
      tags: c.tags,
      status: c.status,
    }))
    send(res, 200, results)
    return
  }

  send(res, 404, { error: 'not found' })
})

const PORT = process.env.MOCK_PORT ?? 3001
server.listen(PORT, () => {
  console.log(`Mock API server running on http://localhost:${PORT}`)
})
