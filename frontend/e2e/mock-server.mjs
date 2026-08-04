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
    has_subtitles: false,
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
    has_subtitles: false,
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
    has_subtitles: false,
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
    has_subtitles: false,
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
    has_subtitles: false,
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
    has_subtitles: false,
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
    has_subtitles: false,
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
  archived_at: null,
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

// Subtitle fixtures, keyed by the admin content id for vid001 (see ADMIN_CONTENTS above)
const SUBTITLE_TRACKS = {
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

const SUBTITLE_CUES = {
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
  ],
}

function send(res, status, data) {
  res.writeHead(status, { 'Content-Type': 'application/json' })
  res.end(JSON.stringify(data))
}

function sendNoContent(res) {
  res.writeHead(204)
  res.end()
}

function readJsonBody(req) {
  return new Promise((resolve, reject) => {
    let raw = ''
    req.on('data', (chunk) => { raw += chunk })
    req.on('end', () => {
      if (!raw) { resolve({}); return }
      try {
        resolve(JSON.parse(raw))
      } catch (e) {
        reject(e)
      }
    })
    req.on('error', reject)
  })
}

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, 'http://localhost')
  const path = url.pathname
  const method = req.method ?? 'GET'

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
  if (path === '/api/v1/admin/contents' && method === 'GET') {
    const limit = parseInt(url.searchParams.get('limit') ?? '200')
    let items = ADMIN_CONTENTS
    const status = url.searchParams.get('status')
    if (status) items = items.filter((c) => c.status === status)
    send(res, 200, items.slice(0, limit))
    return
  }

  // PUT /api/v1/admin/contents/:id
  const updateContentMatch = path.match(/^\/api\/v1\/admin\/contents\/([^/]+)$/)
  if (updateContentMatch && method === 'PUT') {
    const item = ADMIN_CONTENTS.find((c) => c.id === updateContentMatch[1])
    if (!item) { send(res, 404, { error: 'not found' }); return }
    const body = await readJsonBody(req)
    if (body.title != null) item.title = body.title
    if (body.description != null) item.description = body.description
    if (body.tags != null) item.tags = body.tags
    if (body.status != null) item.status = body.status
    item.updated_at = new Date().toISOString()
    send(res, 200, item)
    return
  }

  // POST /api/v1/admin/contents/:id/archive
  const archiveMatch = path.match(/^\/api\/v1\/admin\/contents\/([^/]+)\/archive$/)
  if (archiveMatch && method === 'POST') {
    const item = ADMIN_CONTENTS.find((c) => c.id === archiveMatch[1])
    if (!item) { send(res, 404, { error: 'not found' }); return }
    item.archived_at = new Date().toISOString()
    send(res, 200, item)
    return
  }

  // POST /api/v1/admin/contents/:id/unarchive
  const unarchiveMatch = path.match(/^\/api\/v1\/admin\/contents\/([^/]+)\/unarchive$/)
  if (unarchiveMatch && method === 'POST') {
    const item = ADMIN_CONTENTS.find((c) => c.id === unarchiveMatch[1])
    if (!item) { send(res, 404, { error: 'not found' }); return }
    item.archived_at = null
    send(res, 200, item)
    return
  }

  // GET /api/v1/admin/contents/:contentId/subtitles
  const subtitleTracksMatch = path.match(/^\/api\/v1\/admin\/contents\/([^/]+)\/subtitles$/)
  if (subtitleTracksMatch && method === 'GET') {
    send(res, 200, SUBTITLE_TRACKS[subtitleTracksMatch[1]] ?? [])
    return
  }

  // PUT /api/v1/admin/subtitles/:trackId/status
  const trackStatusMatch = path.match(/^\/api\/v1\/admin\/subtitles\/([^/]+)\/status$/)
  if (trackStatusMatch && method === 'PUT') {
    const track = Object.values(SUBTITLE_TRACKS).flat().find((t) => t.id === trackStatusMatch[1])
    if (!track) { send(res, 404, { error: 'not found' }); return }
    const body = await readJsonBody(req)
    track.status = body.status
    track.updated_at = new Date().toISOString()
    send(res, 200, track)
    return
  }

  // GET /api/v1/admin/subtitles/:trackId/cues
  const cuesMatch = path.match(/^\/api\/v1\/admin\/subtitles\/([^/]+)\/cues$/)
  if (cuesMatch && method === 'GET') {
    send(res, 200, SUBTITLE_CUES[cuesMatch[1]] ?? [])
    return
  }

  // POST /api/v1/admin/subtitles/:trackId/cues
  if (cuesMatch && method === 'POST') {
    const trackId = cuesMatch[1]
    const body = await readJsonBody(req)
    const cue = {
      id: `cue-${Date.now()}`,
      track_id: trackId,
      seq: body.seq,
      start_ms: body.start_ms,
      end_ms: body.end_ms,
      text: body.text,
      original_text: body.text,
      flagged: false,
      updated_at: new Date().toISOString(),
    }
    SUBTITLE_CUES[trackId] = [...(SUBTITLE_CUES[trackId] ?? []), cue]
    send(res, 200, cue)
    return
  }

  // PUT /api/v1/admin/subtitles/cues/:cueId
  const cueUpdateMatch = path.match(/^\/api\/v1\/admin\/subtitles\/cues\/([^/]+)$/)
  if (cueUpdateMatch && method === 'PUT') {
    const cue = Object.values(SUBTITLE_CUES).flat().find((c) => c.id === cueUpdateMatch[1])
    if (!cue) { send(res, 404, { error: 'not found' }); return }
    const body = await readJsonBody(req)
    Object.assign(cue, body, { updated_at: new Date().toISOString() })
    send(res, 200, cue)
    return
  }

  // DELETE /api/v1/admin/subtitles/cues/:cueId
  if (cueUpdateMatch && method === 'DELETE') {
    const cueId = cueUpdateMatch[1]
    let found = false
    for (const trackId of Object.keys(SUBTITLE_CUES)) {
      const before = SUBTITLE_CUES[trackId].length
      SUBTITLE_CUES[trackId] = SUBTITLE_CUES[trackId].filter((c) => c.id !== cueId)
      if (SUBTITLE_CUES[trackId].length !== before) found = true
    }
    if (!found) { send(res, 404, { error: 'not found' }); return }
    sendNoContent(res)
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
