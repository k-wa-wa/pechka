import http from "http";

const CONTENTS = [
  {
    short_id: "abc123",
    content_type: "video",
    title: "サンプル映画 1",
    description: "アクション映画のサンプルコンテンツです。",
    duration_seconds: 7200,
    is_360: false,
    tags: ["映画", "アクション"],
    status: "ready",
    disc_label: "DISC_001",
    thumbnail_key: null,
    published_at: "2026-04-01T00:00:00Z",
    updated_at: "2026-04-01T00:00:00Z",
  },
  {
    short_id: "def456",
    content_type: "vr360",
    title: "360° VR 風景",
    description: "VR コンテンツのサンプルです。",
    duration_seconds: 1800,
    is_360: true,
    tags: ["VR", "風景"],
    status: "ready",
    disc_label: "DISC_002",
    thumbnail_key: null,
    published_at: "2026-04-02T00:00:00Z",
    updated_at: "2026-04-02T00:00:00Z",
  },
  {
    short_id: "ghi789",
    content_type: "video",
    title: "処理中のコンテンツ",
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
    title: "画像ギャラリー",
    description: "写真コレクションです。",
    duration_seconds: null,
    is_360: false,
    tags: ["写真", "風景"],
    status: "ready",
    disc_label: null,
    thumbnail_key: null,
    published_at: "2026-04-04T00:00:00Z",
    updated_at: "2026-04-04T00:00:00Z",
  },
];

const ADMIN_CONTENTS = CONTENTS.map((c, i) => ({
  id: `id-${i + 1}`,
  short_id: c.short_id,
  content_type: c.content_type,
  disc_id: i < 3 ? `disc-${i + 1}` : null,
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

const DISCS = [
  { id: "disc-1", label: "DISC_001", disc_name: "アクション映画コレクション", created_at: "2026-04-01T00:00:00Z" },
  { id: "disc-2", label: "DISC_002", disc_name: "VR コンテンツ集", created_at: "2026-04-02T00:00:00Z" },
  { id: "disc-3", label: "DISC_003", disc_name: null, created_at: "2026-04-03T00:00:00Z" },
];

const server = http.createServer((req, res) => {
  res.setHeader("Content-Type", "application/json");
  res.setHeader("Access-Control-Allow-Origin", "*");

  const url = new URL(req.url, "http://localhost");
  const path = url.pathname;

  if (path === "/v1/contents") {
    const contentType = url.searchParams.get("content_type");
    const filtered = contentType ? CONTENTS.filter((c) => c.content_type === contentType) : CONTENTS;
    res.end(JSON.stringify(filtered));
  } else if (/^\/v1\/contents\/[^/]+\/variants$/.test(path)) {
    res.end(JSON.stringify([]));
  } else if (/^\/v1\/contents\/[^/]+$/.test(path)) {
    const shortId = path.split("/")[3];
    const content = CONTENTS.find((c) => c.short_id === shortId) ?? CONTENTS[0];
    res.end(JSON.stringify(content));
  } else if (path === "/v1/search") {
    const q = url.searchParams.get("q") ?? "";
    const results = q
      ? CONTENTS.filter((c) => c.title.includes(q) || c.tags.some((t) => t.includes(q))).map((c) => ({
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
    res.end(JSON.stringify(ADMIN_CONTENTS));
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
