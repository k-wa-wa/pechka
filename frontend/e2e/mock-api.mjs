import http from "http";

const CONTENTS = [
  {
    short_id: "abc123",
    content_type: "video",
    title: "アクション大作 2026",
    description: "スリリングなアクション映画。スタント満載の超大作。",
    duration_seconds: 7200,
    is_360: false,
    tags: ["映画", "アクション", "4K"],
    status: "ready",
    disc_label: "DISC_001",
    thumbnail_key: null,
    published_at: "2026-04-01T00:00:00Z",
    updated_at: "2026-04-01T00:00:00Z",
  },
  {
    short_id: "def456",
    content_type: "vr360",
    title: "富士山 360° VR 体験",
    description: "山頂からの絶景を360度体験できるVRコンテンツ。",
    duration_seconds: 1800,
    is_360: true,
    tags: ["VR", "風景", "自然"],
    status: "ready",
    disc_label: "DISC_002",
    thumbnail_key: null,
    published_at: "2026-04-02T00:00:00Z",
    updated_at: "2026-04-02T00:00:00Z",
  },
  {
    short_id: "ghi789",
    content_type: "video",
    title: "ドキュメンタリー: 海の旅",
    description: "トランスコード処理中のコンテンツです。",
    duration_seconds: null,
    is_360: false,
    tags: [],
    status: "processing",
    disc_label: "DISC_003",
    thumbnail_key: null,
    published_at: null,
    updated_at: "2026-04-03T00:00:00Z",
  },
  {
    short_id: "jkl012",
    content_type: "image_gallery",
    title: "京都の四季 写真集",
    description: "京都の春夏秋冬を収めた写真コレクション。",
    duration_seconds: null,
    is_360: false,
    tags: ["写真", "風景", "京都"],
    status: "ready",
    disc_label: null,
    thumbnail_key: null,
    published_at: "2026-04-04T00:00:00Z",
    updated_at: "2026-04-04T00:00:00Z",
  },
  {
    short_id: "mno345",
    content_type: "video",
    title: "SF 大作: 宇宙の果て",
    description: "遠い未来の宇宙を舞台にした壮大なSF映画。",
    duration_seconds: 9000,
    is_360: false,
    tags: ["映画", "SF", "宇宙"],
    status: "ready",
    disc_label: "DISC_004",
    thumbnail_key: null,
    published_at: "2026-04-05T00:00:00Z",
    updated_at: "2026-04-05T00:00:00Z",
  },
  {
    short_id: "pqr678",
    content_type: "vr360",
    title: "水中世界 VR",
    description: "サンゴ礁を泳ぐ魚たちを体感する没入型VRコンテンツ。",
    duration_seconds: 2700,
    is_360: true,
    tags: ["VR", "海", "自然"],
    status: "ready",
    disc_label: "DISC_005",
    thumbnail_key: null,
    published_at: "2026-04-06T00:00:00Z",
    updated_at: "2026-04-06T00:00:00Z",
  },
  {
    short_id: "stu901",
    content_type: "video",
    title: "コメディ傑作選",
    description: "笑える短編コメディ映像集。",
    duration_seconds: 3600,
    is_360: false,
    tags: ["映画", "コメディ"],
    status: "ready",
    disc_label: "DISC_006",
    thumbnail_key: null,
    published_at: "2026-04-07T00:00:00Z",
    updated_at: "2026-04-07T00:00:00Z",
  },
  {
    short_id: "vwx234",
    content_type: "document",
    title: "システム設計書 v2",
    description: "pechka システムの技術設計文書。",
    duration_seconds: null,
    is_360: false,
    tags: ["技術", "ドキュメント"],
    status: "ready",
    disc_label: null,
    thumbnail_key: null,
    published_at: "2026-04-08T00:00:00Z",
    updated_at: "2026-04-08T00:00:00Z",
  },
];

const VARIANTS = {
  abc123: [
    { variant_type: "1080p", hls_key: "hls/abc123/1080p/index.m3u8", bandwidth: 6000000, resolution: "1920x1080", codecs: "avc1.64001f,mp4a.40.2" },
    { variant_type: "720p", hls_key: "hls/abc123/720p/index.m3u8", bandwidth: 3000000, resolution: "1280x720", codecs: "avc1.64001f,mp4a.40.2" },
    { variant_type: "480p", hls_key: "hls/abc123/480p/index.m3u8", bandwidth: 1500000, resolution: "854x480", codecs: "avc1.64001f,mp4a.40.2" },
    { variant_type: "audio", hls_key: "hls/abc123/audio/index.m3u8", bandwidth: 192000, resolution: null, codecs: "mp4a.40.2" },
  ],
  def456: [
    { variant_type: "1080p", hls_key: "hls/def456/1080p/index.m3u8", bandwidth: 6000000, resolution: "1920x1080", codecs: "avc1.64001f,mp4a.40.2" },
    { variant_type: "720p", hls_key: "hls/def456/720p/index.m3u8", bandwidth: 3000000, resolution: "1280x720", codecs: "avc1.64001f,mp4a.40.2" },
  ],
  mno345: [
    { variant_type: "1080p", hls_key: "hls/mno345/1080p/index.m3u8", bandwidth: 6000000, resolution: "1920x1080", codecs: "avc1.64001f,mp4a.40.2" },
    { variant_type: "720p", hls_key: "hls/mno345/720p/index.m3u8", bandwidth: 3000000, resolution: "1280x720", codecs: "avc1.64001f,mp4a.40.2" },
  ],
  pqr678: [
    { variant_type: "1080p", hls_key: "hls/pqr678/1080p/index.m3u8", bandwidth: 6000000, resolution: "1920x1080", codecs: "avc1.64001f,mp4a.40.2" },
  ],
  stu901: [
    { variant_type: "720p", hls_key: "hls/stu901/720p/index.m3u8", bandwidth: 3000000, resolution: "1280x720", codecs: "avc1.64001f,mp4a.40.2" },
  ],
};

const ADMIN_CONTENTS = CONTENTS.map((c, i) => ({
  id: `id-${i + 1}`,
  short_id: c.short_id,
  content_type: c.content_type,
  disc_id: c.disc_label ? `disc-${i + 1}` : null,
  title: c.title,
  description: c.description,
  duration_seconds: c.duration_seconds,
  is_360: c.is_360,
  tags: c.tags,
  status: c.status,
  published_at: c.published_at,
  created_at: "2026-04-01T00:00:00Z",
  updated_at: c.updated_at,
}));

const DISCS = CONTENTS
  .filter((c) => c.disc_label)
  .map((c, i) => ({
    id: `disc-${i + 1}`,
    label: c.disc_label,
    disc_name: c.title,
    created_at: c.published_at ?? "2026-04-01T00:00:00Z",
  }));

const server = http.createServer((req, res) => {
  res.setHeader("Content-Type", "application/json");
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS");
  res.setHeader("Access-Control-Allow-Headers", "Content-Type");

  if (req.method === "OPTIONS") {
    res.writeHead(204);
    res.end();
    return;
  }

  const url = new URL(req.url, "http://localhost");
  const path = url.pathname;

  if (path === "/v1/contents") {
    const contentType = url.searchParams.get("content_type");
    const filtered = contentType ? CONTENTS.filter((c) => c.content_type === contentType) : CONTENTS;
    res.end(JSON.stringify(filtered));
  } else if (/^\/v1\/contents\/[^/]+\/variants$/.test(path)) {
    const shortId = path.split("/")[3];
    res.end(JSON.stringify(VARIANTS[shortId] ?? []));
  } else if (/^\/v1\/contents\/[^/]+$/.test(path)) {
    const shortId = path.split("/")[3];
    const content = CONTENTS.find((c) => c.short_id === shortId) ?? CONTENTS[0];
    res.end(JSON.stringify(content));
  } else if (path === "/v1/search") {
    const q = url.searchParams.get("q") ?? "";
    const results = q
      ? CONTENTS.filter(
          (c) => c.title.includes(q) || c.tags.some((t) => t.includes(q))
        ).map((c) => ({
          short_id: c.short_id,
          title: c.title,
          description: c.description,
          content_type: c.content_type,
          tags: c.tags,
          status: c.status,
        }))
      : [];
    res.end(JSON.stringify(results));
  } else if (path === "/v1/admin/contents") {
    if (req.method === "POST") {
      res.statusCode = 405;
      res.end(JSON.stringify({ error: "Content creation is handled by ETL pipeline" }));
      return;
    }
    res.end(JSON.stringify(ADMIN_CONTENTS));
  } else if (/^\/v1\/admin\/contents\/[^/]+$/.test(path)) {
    if (req.method === "PUT") {
      res.end(JSON.stringify({ ok: true }));
      return;
    }
    if (req.method === "DELETE") {
      res.writeHead(204);
      res.end();
      return;
    }
    res.statusCode = 404;
    res.end(JSON.stringify({ error: "not found" }));
  } else if (path === "/v1/admin/discs") {
    res.end(JSON.stringify(DISCS));
  } else {
    res.statusCode = 404;
    res.end(JSON.stringify({ error: "not found" }));
  }
});

server.listen(9999, () => {
  console.log("Mock API listening on :9999");
});
