import http from "node:http";

const CHANNELS = [
  {
    id: "tech-feed",
    name: "Tech Feed",
    description: "自動収集・AI生成による技術ダイジェスト解説",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  },
  {
    id: "bluray",
    name: "Blu-ray",
    description: "Blu-ray取り込み動画",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  },
];

const CONTENTS = [
  {
    short_id: "vid001",
    content_type: "video",
    channel_id: "bluray",
    channel_name: "Blu-ray",
    title: "夏の思い出 2024",
    description: "海辺での夏のひとこまを収めた映像です。",
    duration_seconds: 342,
    is_360: false,
    tags: ["夏", "海", "思い出"],
    status: "ready",
    disc_label: "DISC-001",
    thumbnail_key: null,
    published_at: "2024-08-15T00:00:00Z",
    updated_at: "2024-08-20T10:00:00Z",
    has_subtitles: false,
  },
  {
    short_id: "vid002",
    content_type: "video",
    channel_id: "bluray",
    channel_name: "Blu-ray",
    title: "秋の紅葉ドライブ",
    description: "山道を走りながら撮影した紅葉の映像。",
    duration_seconds: 580,
    is_360: false,
    tags: ["秋", "紅葉", "ドライブ"],
    status: "ready",
    disc_label: null,
    thumbnail_key: null,
    published_at: "2024-11-03T00:00:00Z",
    updated_at: "2024-11-05T08:00:00Z",
    has_subtitles: false,
  },
  {
    short_id: "gal001",
    content_type: "image_gallery",
    channel_id: "bluray",
    channel_name: "Blu-ray",
    title: "春の花々フォトギャラリー",
    description: "桜・菜の花・チューリップなど春の花の写真集。",
    duration_seconds: null,
    is_360: false,
    tags: ["春", "花", "写真"],
    status: "ready",
    disc_label: null,
    thumbnail_key: null,
    published_at: "2024-04-01T00:00:00Z",
    updated_at: "2024-04-02T09:00:00Z",
    has_subtitles: false,
  },
  {
    short_id: "vr001",
    content_type: "vr360",
    channel_id: "bluray",
    channel_name: "Blu-ray",
    title: "360° 富士山山頂の眺め",
    description: "富士山山頂からの360度パノラマ映像。",
    duration_seconds: 120,
    is_360: true,
    tags: ["富士山", "VR", "360度"],
    status: "ready",
    disc_label: null,
    thumbnail_key: null,
    published_at: "2024-07-20T00:00:00Z",
    updated_at: "2024-07-21T12:00:00Z",
    has_subtitles: false,
  },
  {
    short_id: "vid003",
    content_type: "video",
    channel_id: "bluray",
    channel_name: "Blu-ray",
    title: "冬の北海道旅行記",
    description: "雪景色の北海道を旅した記録映像。",
    duration_seconds: 1240,
    is_360: false,
    tags: ["冬", "北海道", "旅行"],
    status: "ready",
    disc_label: "DISC-002",
    thumbnail_key: null,
    published_at: "2024-01-10T00:00:00Z",
    updated_at: "2024-01-15T14:00:00Z",
    has_subtitles: false,
  },
  {
    short_id: "doc001",
    content_type: "document",
    channel_id: "tech-feed",
    channel_name: "Tech Feed",
    title: "2024年度 活動報告書",
    description: "年間の活動をまとめたドキュメント。",
    duration_seconds: null,
    is_360: false,
    tags: ["報告", "ドキュメント"],
    status: "ready",
    disc_label: null,
    thumbnail_key: null,
    document_key: "resources/documents/doc001/document.md",
    document_mime_type: "text/markdown",
    published_at: "2025-01-05T00:00:00Z",
    updated_at: "2025-01-06T09:00:00Z",
    has_subtitles: false,
  },
  {
    short_id: "vid004",
    content_type: "video",
    channel_id: "bluray",
    channel_name: "Blu-ray",
    title: "処理中のコンテンツ",
    description: "",
    duration_seconds: null,
    is_360: false,
    tags: [],
    status: "processing",
    disc_label: null,
    thumbnail_key: null,
    published_at: null,
    updated_at: "2025-04-30T10:00:00Z",
    has_subtitles: false,
  },
];

const ADMIN_CONTENTS = CONTENTS.map((c, i) => ({
  id: `00000000-0000-0000-0000-${String(i + 1).padStart(12, "0")}`,
  short_id: c.short_id,
  content_type: c.content_type,
  channel_id: c.channel_id,
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
}));

const SAMPLE_MARKDOWN = `# 2024年度 活動報告書

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
`;

const VARIANTS = {
  vid001: [
    {
      variant_type: "master",
      hls_key: "vid001/master.m3u8",
      bandwidth: null,
      resolution: null,
      codecs: null,
    },
    {
      variant_type: "1080p",
      hls_key: "vid001/1080p/index.m3u8",
      bandwidth: 5000000,
      resolution: "1920x1080",
      codecs: "avc1.640028,mp4a.40.2",
    },
    {
      variant_type: "720p",
      hls_key: "vid001/720p/index.m3u8",
      bandwidth: 2800000,
      resolution: "1280x720",
      codecs: "avc1.4d401f,mp4a.40.2",
    },
    {
      variant_type: "480p",
      hls_key: "vid001/480p/index.m3u8",
      bandwidth: 1400000,
      resolution: "854x480",
      codecs: "avc1.4d401e,mp4a.40.2",
    },
  ],
  vid002: [
    {
      variant_type: "master",
      hls_key: "vid002/master.m3u8",
      bandwidth: null,
      resolution: null,
      codecs: null,
    },
    {
      variant_type: "720p",
      hls_key: "vid002/720p/index.m3u8",
      bandwidth: 2800000,
      resolution: "1280x720",
      codecs: "avc1.4d401f,mp4a.40.2",
    },
  ],
  vr001: [
    {
      variant_type: "master",
      hls_key: "vr001/master.m3u8",
      bandwidth: null,
      resolution: null,
      codecs: null,
    },
    {
      variant_type: "1080p",
      hls_key: "vr001/1080p/index.m3u8",
      bandwidth: 5000000,
      resolution: "3840x1920",
      codecs: "avc1.640028,mp4a.40.2",
    },
  ],
};

// Subtitle fixtures, keyed by the admin content id for vid001 (see ADMIN_CONTENTS above)
const SUBTITLE_TRACKS = {
  "00000000-0000-0000-0000-000000000001": [
    {
      id: "track-vid001-ja",
      content_id: "00000000-0000-0000-0000-000000000001",
      language: "ja",
      status: "published",
      model: "whisper-large-v3",
      created_at: "2024-08-20T10:00:00Z",
      updated_at: "2024-08-20T10:05:00Z",
    },
  ],
};

const SUBTITLE_CUES = {
  "track-vid001-ja": [
    {
      id: "cue-1",
      track_id: "track-vid001-ja",
      seq: 1,
      start_ms: 0,
      end_ms: 3200,
      text: "夏の海辺にやってきました。",
      original_text: "夏の海辺にやってきました。",
      flagged: false,
      updated_at: "2024-08-20T10:05:00Z",
    },
    {
      id: "cue-2",
      track_id: "track-vid001-ja",
      seq: 2,
      start_ms: 3200,
      end_ms: 6800,
      text: "波の音がとても心地よいですね。",
      original_text: "波の音がとてもここちよいですね。",
      flagged: true,
      updated_at: "2024-08-20T10:05:00Z",
    },
  ],
};

const CORS_HEADERS = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
  "Access-Control-Allow-Headers": "Content-Type, X-Channel-Id",
};

function send(res, status, data) {
  res.writeHead(status, { "Content-Type": "application/json", ...CORS_HEADERS });
  res.end(JSON.stringify(data));
}

function sendNoContent(res) {
  res.writeHead(204, CORS_HEADERS);
  res.end();
}

function readJsonBody(req) {
  return new Promise((resolve, reject) => {
    let raw = "";
    req.on("data", (chunk) => {
      raw += chunk;
    });
    req.on("end", () => {
      if (!raw) {
        resolve({});
        return;
      }
      try {
        resolve(JSON.parse(raw));
      } catch (e) {
        reject(e);
      }
    });
    req.on("error", reject);
  });
}

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, "http://localhost");
  const path = url.pathname;
  const method = req.method ?? "GET";

  if (method === "OPTIONS") {
    res.writeHead(204, CORS_HEADERS);
    res.end();
    return;
  }

  // GET /api/v1/channels
  if (path === "/api/v1/channels") {
    send(res, 200, CHANNELS);
    return;
  }

  // GET /api/v1/channels/:id
  const channelMatch = path.match(/^\/api\/v1\/channels\/([^/]+)$/);
  if (channelMatch) {
    const ch = CHANNELS.find((c) => c.id === channelMatch[1]);
    if (!ch) {
      send(res, 404, { error: "not found" });
      return;
    }
    send(res, 200, ch);
    return;
  }

  // GET /api/v1/contents
  if (path === "/api/v1/contents") {
    const contentType = url.searchParams.get("content_type");
    const channelId = url.searchParams.get("channel_id");
    const limit = parseInt(url.searchParams.get("limit") ?? "100");
    let items = CONTENTS;
    if (contentType) items = items.filter((c) => c.content_type === contentType);
    if (channelId) items = items.filter((c) => c.channel_id === channelId);
    send(res, 200, items.slice(0, limit));
    return;
  }

  // GET /api/v1/contents/:shortId
  const contentMatch = path.match(/^\/api\/v1\/contents\/([^/]+)$/);
  if (contentMatch) {
    const item = CONTENTS.find((c) => c.short_id === contentMatch[1]);
    if (!item) {
      send(res, 404, { error: "not found" });
      return;
    }
    send(res, 200, item);
    return;
  }

  // GET /api/v1/contents/:shortId/variants
  const variantsMatch = path.match(/^\/api\/v1\/contents\/([^/]+)\/variants$/);
  if (variantsMatch) {
    const variants = VARIANTS[variantsMatch[1]] ?? [];
    send(res, 200, variants);
    return;
  }

  // GET /resources/documents/:shortId/:filename
  const docResourceMatch = path.match(/^\/resources\/documents\/([^/]+)\/([^/]+)$/);
  if (docResourceMatch && method === "GET") {
    const filename = docResourceMatch[2];
    if (filename.endsWith(".md") || filename.endsWith(".markdown")) {
      res.writeHead(200, {
        "Content-Type": "text/markdown; charset=utf-8",
        "X-Content-Type-Options": "nosniff",
        ...CORS_HEADERS,
      });
      res.end(SAMPLE_MARKDOWN);
      return;
    }
    if (filename.endsWith(".pdf")) {
      res.writeHead(200, {
        "Content-Type": "application/pdf",
        "X-Content-Type-Options": "nosniff",
        ...CORS_HEADERS,
      });
      res.end("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n%%EOF\n");
      return;
    }
    send(res, 404, { error: "not found" });
    return;
  }

  // GET /api/v1/admin/contents
  if (path === "/api/v1/admin/contents" && method === "GET") {
    const limit = parseInt(url.searchParams.get("limit") ?? "20");
    const offset = parseInt(url.searchParams.get("offset") ?? "0");
    let items = ADMIN_CONTENTS;
    const status = url.searchParams.get("status");
    if (status) items = items.filter((c) => c.status === status);
    send(res, 200, {
      contents: items.slice(offset, offset + limit),
      total: items.length,
      limit,
      offset,
    });
    return;
  }

  // PUT /api/v1/admin/contents/:id
  const updateContentMatch = path.match(/^\/api\/v1\/admin\/contents\/([^/]+)$/);
  if (updateContentMatch && method === "PUT") {
    const item = ADMIN_CONTENTS.find((c) => c.id === updateContentMatch[1]);
    if (!item) {
      send(res, 404, { error: "not found" });
      return;
    }
    const body = await readJsonBody(req);
    if (body.channel_id != null) item.channel_id = body.channel_id;
    if (body.title != null) item.title = body.title;
    if (body.description != null) item.description = body.description;
    if (body.tags != null) item.tags = body.tags;
    if (body.status != null) item.status = body.status;
    item.updated_at = new Date().toISOString();

    // Sync to public CONTENTS mock so changes are immediately visible on / and /channels/:id
    const pubItem = CONTENTS.find((c) => c.short_id === item.short_id);
    if (pubItem) {
      if (body.channel_id != null) {
        pubItem.channel_id = body.channel_id;
        const ch = CHANNELS.find((c) => c.id === body.channel_id);
        pubItem.channel_name = ch ? ch.name : body.channel_id;
      }
      if (body.title != null) pubItem.title = body.title;
      if (body.description != null) pubItem.description = body.description;
      if (body.tags != null) pubItem.tags = body.tags;
      if (body.status != null) pubItem.status = body.status;
      pubItem.updated_at = item.updated_at;
    }
    send(res, 200, item);
    return;
  }

  // POST /api/v1/admin/contents/upload
  // POST /api/v1/admin/contents/upload
  if (path === "/api/v1/admin/contents/upload" && method === "POST") {
    const chunks = [];
    for await (const chunk of req) chunks.push(chunk);
    const body = Buffer.concat(chunks).toString("utf8");
    const titleMatch = body.match(/name="title"\r\n\r\n(.*)\r\n/);
    const title = titleMatch ? titleMatch[1] : null;
    if (!title) {
      send(res, 400, { error: "file and title are required" });
      return;
    }
    const channelId = (req.headers["x-channel-id"] || "").trim();
    if (!channelId) {
      send(res, 400, { error: "channel_id is required" });
      return;
    }
    const filenameMatch = body.match(/filename="([^"]+)"/);
    const uploadedFilename = (filenameMatch ? filenameMatch[1] : "").toLowerCase();
    const isMarkdown = uploadedFilename.endsWith(".md") || uploadedFilename.endsWith(".markdown");
    const isPdf = uploadedFilename.endsWith(".pdf");
    const isDoc = isMarkdown || isPdf;
    const now = new Date().toISOString();
    const shortId = `up${Date.now()}`;
    const docKey = isMarkdown
      ? `resources/documents/${shortId}/document.md`
      : isPdf
        ? `resources/documents/${shortId}/document.pdf`
        : null;
    const docMime = isMarkdown ? "text/markdown" : isPdf ? "application/pdf" : null;
    const content = {
      id: `upload-${Date.now()}`,
      short_id: shortId,
      content_type: isDoc ? "document" : "video",
      channel_id: channelId,
      disc_id: null,
      title,
      description: "",
      duration_seconds: null,
      is_360: false,
      tags: [],
      status: isDoc ? "ready" : "pending",
      document_key: docKey,
      document_mime_type: docMime,
      published_at: isDoc ? now : null,
      archived_at: null,
      created_at: now,
      updated_at: now,
    };
    ADMIN_CONTENTS.unshift(content);
    if (isDoc) {
      const ch = CHANNELS.find((c) => c.id === channelId);
      CONTENTS.unshift({
        short_id: shortId,
        content_type: "document",
        channel_id: channelId,
        channel_name: ch ? ch.name : channelId,
        title,
        description: "",
        duration_seconds: null,
        is_360: false,
        tags: [],
        status: "ready",
        disc_label: null,
        thumbnail_key: null,
        document_key: docKey,
        document_mime_type: docMime,
        published_at: now,
        updated_at: now,
        has_subtitles: false,
      });
    }
    send(res, 201, { content, workflow_name: isDoc ? "" : "video-upload-mock" });
    return;
  }

  // POST /api/v1/admin/contents/:id/archive
  const archiveMatch = path.match(/^\/api\/v1\/admin\/contents\/([^/]+)\/archive$/);
  if (archiveMatch && method === "POST") {
    const item = ADMIN_CONTENTS.find((c) => c.id === archiveMatch[1]);
    if (!item) {
      send(res, 404, { error: "not found" });
      return;
    }
    item.archived_at = new Date().toISOString();
    send(res, 200, item);
    return;
  }

  // POST /api/v1/admin/contents/:id/unarchive
  const unarchiveMatch = path.match(/^\/api\/v1\/admin\/contents\/([^/]+)\/unarchive$/);
  if (unarchiveMatch && method === "POST") {
    const item = ADMIN_CONTENTS.find((c) => c.id === unarchiveMatch[1]);
    if (!item) {
      send(res, 404, { error: "not found" });
      return;
    }
    item.archived_at = null;
    send(res, 200, item);
    return;
  }

  // GET /api/v1/admin/contents/:contentId/subtitles
  const subtitleTracksMatch = path.match(/^\/api\/v1\/admin\/contents\/([^/]+)\/subtitles$/);
  if (subtitleTracksMatch && method === "GET") {
    send(res, 200, SUBTITLE_TRACKS[subtitleTracksMatch[1]] ?? []);
    return;
  }

  // PUT /api/v1/admin/subtitles/:trackId/status
  const trackStatusMatch = path.match(/^\/api\/v1\/admin\/subtitles\/([^/]+)\/status$/);
  if (trackStatusMatch && method === "PUT") {
    const track = Object.values(SUBTITLE_TRACKS)
      .flat()
      .find((t) => t.id === trackStatusMatch[1]);
    if (!track) {
      send(res, 404, { error: "not found" });
      return;
    }
    const body = await readJsonBody(req);
    track.status = body.status;
    track.updated_at = new Date().toISOString();
    send(res, 200, track);
    return;
  }

  // GET /api/v1/admin/subtitles/:trackId/cues
  const cuesMatch = path.match(/^\/api\/v1\/admin\/subtitles\/([^/]+)\/cues$/);
  if (cuesMatch && method === "GET") {
    send(res, 200, SUBTITLE_CUES[cuesMatch[1]] ?? []);
    return;
  }

  // POST /api/v1/admin/subtitles/:trackId/cues
  if (cuesMatch && method === "POST") {
    const trackId = cuesMatch[1];
    const body = await readJsonBody(req);
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
    };
    SUBTITLE_CUES[trackId] = [...(SUBTITLE_CUES[trackId] ?? []), cue];
    send(res, 200, cue);
    return;
  }

  // PUT /api/v1/admin/subtitles/cues/:cueId
  const cueUpdateMatch = path.match(/^\/api\/v1\/admin\/subtitles\/cues\/([^/]+)$/);
  if (cueUpdateMatch && method === "PUT") {
    const cue = Object.values(SUBTITLE_CUES)
      .flat()
      .find((c) => c.id === cueUpdateMatch[1]);
    if (!cue) {
      send(res, 404, { error: "not found" });
      return;
    }
    const body = await readJsonBody(req);
    Object.assign(cue, body, { updated_at: new Date().toISOString() });
    send(res, 200, cue);
    return;
  }

  // DELETE /api/v1/admin/subtitles/cues/:cueId
  if (cueUpdateMatch && method === "DELETE") {
    const cueId = cueUpdateMatch[1];
    let found = false;
    for (const trackId of Object.keys(SUBTITLE_CUES)) {
      const before = SUBTITLE_CUES[trackId].length;
      SUBTITLE_CUES[trackId] = SUBTITLE_CUES[trackId].filter((c) => c.id !== cueId);
      if (SUBTITLE_CUES[trackId].length !== before) found = true;
    }
    if (!found) {
      send(res, 404, { error: "not found" });
      return;
    }
    sendNoContent(res);
    return;
  }

  // GET /api/v1/search
  if (path === "/api/v1/search") {
    const q = (url.searchParams.get("q") ?? "").toLowerCase();
    const results = CONTENTS.filter(
      (c) =>
        c.title.toLowerCase().includes(q) ||
        c.description.toLowerCase().includes(q) ||
        c.tags.some((t) => t.toLowerCase().includes(q)),
    ).map((c) => ({
      short_id: c.short_id,
      title: c.title,
      description: c.description,
      content_type: c.content_type,
      tags: c.tags,
      status: c.status,
    }));
    send(res, 200, results);
    return;
  }

  send(res, 404, { error: "not found" });
});

const PORT = process.env.MOCK_PORT ?? 3001;
const HOST = process.env.MOCK_HOST ?? "127.0.0.1";
server.listen(PORT, HOST, () => {
  console.log(`Mock API server running on http://${HOST}:${PORT}`);
});
