import { realpathSync } from "node:fs";
import http from "node:http";
import { pathToFileURL } from "node:url";

// 画面・スナップショットに出る時刻を固定する（VRT の差分を防ぐ）。モックの既存データより後の日時にする
const MOCK_NOW = "2025-06-01T00:00:00Z";

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
  deleted_at: null,
  created_at: c.updated_at,
  updated_at: c.updated_at,
}));

const VALID_ADMIN_SORTS = new Set([
  "updated_desc",
  "updated_asc",
  "created_desc",
  "created_asc",
  "title_asc",
  "title_desc",
]);

function compareCodePoints(a, b) {
  if (a === b) return 0;
  const iterA = a[Symbol.iterator]();
  const iterB = b[Symbol.iterator]();
  while (true) {
    const nextA = iterA.next();
    const nextB = iterB.next();
    if (nextA.done && nextB.done) return 0;
    if (nextA.done) return -1;
    if (nextB.done) return 1;
    const cpA = nextA.value.codePointAt(0);
    const cpB = nextB.value.codePointAt(0);
    if (cpA !== cpB) return cpA < cpB ? -1 : 1;
  }
}

function sortMeta(sort) {
  switch (sort) {
    case "updated_desc":
      return { field: "updated_at", dir: -1, isTitle: false };
    case "updated_asc":
      return { field: "updated_at", dir: 1, isTitle: false };
    case "created_desc":
      return { field: "created_at", dir: -1, isTitle: false };
    case "created_asc":
      return { field: "created_at", dir: 1, isTitle: false };
    case "title_asc":
      return { field: "title", dir: 1, isTitle: true };
    case "title_desc":
      return { field: "title", dir: -1, isTitle: true };
    default:
      return null;
  }
}

function encodeAdminCursor(sort, item) {
  const meta = sortMeta(sort);
  const v = meta.isTitle ? item.title : Date.parse(item[meta.field]);
  const json = JSON.stringify({ sort, v, id: item.short_id });
  return Buffer.from(json, "utf8").toString("base64url");
}

const INT64_MAX_DIGITS = "9223372036854775807";
const INT64_MIN_ABS_DIGITS = "9223372036854775808";

function extractRawJsonObjectField(jsonText, targetKey) {
  let i = 0;
  const n = jsonText.length;
  const skipWs = () => {
    while (
      i < n &&
      (jsonText[i] === " " || jsonText[i] === "\t" || jsonText[i] === "\r" || jsonText[i] === "\n")
    ) {
      i++;
    }
  };
  const readString = () => {
    if (jsonText[i] !== '"') return null;
    const start = i++;
    while (i < n) {
      const ch = jsonText[i++];
      if (ch === "\\") {
        if (i >= n) return null;
        i++;
      } else if (ch === '"') {
        return jsonText.slice(start, i);
      }
    }
    return null;
  };
  const skipValue = () => {
    skipWs();
    if (i >= n) return null;
    const start = i;
    const ch = jsonText[i];
    if (ch === '"') {
      if (readString() === null) return null;
      return jsonText.slice(start, i);
    }
    if (ch === "{" || ch === "[") {
      const open = ch;
      const close = ch === "{" ? "}" : "]";
      let depth = 0;
      while (i < n) {
        const c = jsonText[i];
        if (c === '"') {
          if (readString() === null) return null;
        } else if (c === open) {
          depth++;
          i++;
        } else if (c === close) {
          depth--;
          i++;
          if (depth === 0) return jsonText.slice(start, i);
        } else {
          i++;
        }
      }
      return null;
    }
    while (i < n && !",}] \t\r\n".includes(jsonText[i])) {
      i++;
    }
    return jsonText.slice(start, i);
  };

  skipWs();
  if (jsonText[i] !== "{") return null;
  i++;
  let found = null;
  while (i < n) {
    skipWs();
    if (jsonText[i] === "}") break;
    const rawKey = readString();
    if (rawKey === null) return null;
    const key = JSON.parse(rawKey);
    skipWs();
    if (jsonText[i] !== ":") return null;
    i++;
    const rawVal = skipValue();
    if (rawVal === null) return null;
    if (key === targetKey) {
      found = rawVal;
    }
    skipWs();
    if (jsonText[i] === ",") {
      i++;
    } else if (jsonText[i] === "}") {
      break;
    } else {
      return null;
    }
  }
  return found;
}

function parseStrictInt64Token(rawToken) {
  if (rawToken === null || !/^-?(?:0|[1-9][0-9]*)$/.test(rawToken)) {
    return null;
  }
  const negative = rawToken.startsWith("-");
  const digits = negative ? rawToken.slice(1) : rawToken;
  const maxDigits = negative ? INT64_MIN_ABS_DIGITS : INT64_MAX_DIGITS;
  if (
    digits.length > maxDigits.length ||
    (digits.length === maxDigits.length && digits > maxDigits)
  ) {
    return null;
  }
  const num = Number(rawToken);
  return num === 0 ? 0 : num;
}

function decodeAdminCursor(token, expectedSort) {
  if (!token || !/^[A-Za-z0-9_-]+$/.test(token) || token.length % 4 === 1) return null;
  try {
    const decoded = Buffer.from(token, "base64url").toString("utf8");
    const parsed = JSON.parse(decoded);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return null;
    if (!VALID_ADMIN_SORTS.has(parsed.sort) || parsed.sort !== expectedSort) return null;
    if (typeof parsed.id !== "string" || parsed.id === "") return null;
    const meta = sortMeta(parsed.sort);
    if (meta.isTitle) {
      if (typeof parsed.v !== "string") return null;
    } else {
      const rawV = extractRawJsonObjectField(decoded, "v");
      const int64V = parseStrictInt64Token(rawV);
      if (int64V === null) return null;
      parsed.v = int64V;
    }
    return parsed;
  } catch {
    return null;
  }
}

function compareAdminItems(a, b, sort) {
  const meta = sortMeta(sort);
  let cmp = 0;
  if (meta.isTitle) {
    cmp = compareCodePoints(a.title, b.title);
  } else {
    const ta = Date.parse(a[meta.field]);
    const tb = Date.parse(b[meta.field]);
    cmp = ta === tb ? 0 : ta < tb ? -1 : 1;
  }
  if (cmp !== 0) return cmp * meta.dir;
  return compareCodePoints(a.short_id, b.short_id);
}

function compareItemToCursor(item, cursor, sort) {
  const meta = sortMeta(sort);
  let cmp = 0;
  if (meta.isTitle) {
    cmp = compareCodePoints(item.title, cursor.v);
  } else {
    const ti = Date.parse(item[meta.field]);
    cmp = ti === cursor.v ? 0 : ti < cursor.v ? -1 : 1;
  }
  if (cmp !== 0) return cmp * meta.dir;
  return compareCodePoints(item.short_id, cursor.id);
}

function stripDeletedField(item) {
  const { deleted_at: _deletedAt, ...rest } = item;
  return rest;
}

function isContentDeleted(contentId) {
  const item = ADMIN_CONTENTS.find((c) => c.id === contentId);
  return item != null && item.deleted_at != null;
}

function findActiveTrack(trackId) {
  for (const tracks of Object.values(SUBTITLE_TRACKS)) {
    const track = tracks.find((t) => t.id === trackId);
    if (track) {
      return isContentDeleted(track.content_id) ? null : track;
    }
  }
  return null;
}

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

const VALID_CHANNEL_ID_RE = /^[a-z0-9][a-z0-9-]{0,49}$/;

function parseStrictLimit(rawParam) {
  if (rawParam === null || !/^[+-]?\d+$/.test(rawParam)) {
    return 20;
  }
  const parsed = Number(rawParam);
  return Number.isSafeInteger(parsed) && parsed >= 1 && parsed <= 100 ? parsed : 20;
}

export function createMockServer() {
  return http.createServer(async (req, res) => {
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
      let items = CONTENTS.filter((c) => !c.deleted_at);
      if (contentType) items = items.filter((c) => c.content_type === contentType);
      if (channelId) items = items.filter((c) => c.channel_id === channelId);
      send(res, 200, items.slice(0, limit));
      return;
    }

    // GET /api/v1/contents/:shortId
    const contentMatch = path.match(/^\/api\/v1\/contents\/([^/]+)$/);
    if (contentMatch) {
      const item = CONTENTS.find((c) => c.short_id === contentMatch[1] && !c.deleted_at);
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
      const item = CONTENTS.find((c) => c.short_id === variantsMatch[1] && !c.deleted_at);
      if (!item) {
        send(res, 404, { error: "not found" });
        return;
      }
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
      const limit = parseStrictLimit(url.searchParams.get("limit"));
      const channelId = url.searchParams.get("channel_id");
      if (channelId && !VALID_CHANNEL_ID_RE.test(channelId)) {
        send(res, 400, { message: "invalid channel_id" });
        return;
      }
      const sort = url.searchParams.get("sort") || "updated_desc";
      if (!VALID_ADMIN_SORTS.has(sort)) {
        send(res, 400, { message: "invalid sort" });
        return;
      }
      const afterParam = url.searchParams.get("after") ?? "";
      const beforeParam = url.searchParams.get("before") ?? "";
      if (afterParam && beforeParam) {
        send(res, 400, { message: "after and before cannot be specified together" });
        return;
      }
      const afterCur = afterParam ? decodeAdminCursor(afterParam, sort) : null;
      const beforeCur = beforeParam ? decodeAdminCursor(beforeParam, sort) : null;
      if ((afterParam && !afterCur) || (beforeParam && !beforeCur)) {
        send(res, 400, { message: "invalid cursor" });
        return;
      }

      let items = ADMIN_CONTENTS.filter((c) => c.deleted_at == null);
      const status = url.searchParams.get("status");
      if (status) items = items.filter((c) => c.status === status);
      if (channelId) items = items.filter((c) => c.channel_id === channelId);

      const sorted = [...items].sort((a, b) => compareAdminItems(a, b, sort));
      const total = sorted.length;

      let pageItems = [];
      let hasNext = false;
      let hasPrev = false;

      if (afterCur) {
        const afterMatches = sorted.filter((c) => compareItemToCursor(c, afterCur, sort) > 0);
        pageItems = afterMatches.slice(0, limit);
        if (pageItems.length > 0) {
          hasNext = afterMatches.length > limit;
          const firstCur = {
            sort,
            v: sortMeta(sort).isTitle
              ? pageItems[0].title
              : Date.parse(pageItems[0][sortMeta(sort).field]),
            id: pageItems[0].short_id,
          };
          hasPrev = sorted.some((c) => compareItemToCursor(c, firstCur, sort) < 0);
        }
      } else if (beforeCur) {
        const beforeMatches = sorted.filter((c) => compareItemToCursor(c, beforeCur, sort) < 0);
        hasPrev = beforeMatches.length > limit;
        pageItems = beforeMatches.slice(-limit);
        if (pageItems.length > 0) {
          const last = pageItems[pageItems.length - 1];
          const lastCur = {
            sort,
            v: sortMeta(sort).isTitle ? last.title : Date.parse(last[sortMeta(sort).field]),
            id: last.short_id,
          };
          hasNext = sorted.some((c) => compareItemToCursor(c, lastCur, sort) > 0);
        }
      } else {
        pageItems = sorted.slice(0, limit);
        hasNext = sorted.length > limit;
        hasPrev = false;
      }

      const nextCursor =
        hasNext && pageItems.length > 0
          ? encodeAdminCursor(sort, pageItems[pageItems.length - 1])
          : null;
      const prevCursor =
        hasPrev && pageItems.length > 0 ? encodeAdminCursor(sort, pageItems[0]) : null;

      send(res, 200, {
        contents: pageItems.map(stripDeletedField),
        total,
        limit,
        next_cursor: nextCursor,
        prev_cursor: prevCursor,
        has_next: hasNext,
        has_prev: hasPrev,
      });
      return;
    }

    // DELETE /api/v1/admin/contents/:id
    const deleteContentMatch = path.match(/^\/api\/v1\/admin\/contents\/([^/]+)$/);
    if (deleteContentMatch && method === "DELETE") {
      const item = ADMIN_CONTENTS.find((c) => c.id === deleteContentMatch[1]);
      if (item && !item.deleted_at) {
        item.deleted_at = MOCK_NOW;
        item.updated_at = MOCK_NOW;
        const pubItem = CONTENTS.find((c) => c.short_id === item.short_id);
        if (pubItem) pubItem.deleted_at = MOCK_NOW;
      }
      sendNoContent(res);
      return;
    }

    // PUT /api/v1/admin/contents/:id
    const updateContentMatch = path.match(/^\/api\/v1\/admin\/contents\/([^/]+)$/);
    if (updateContentMatch && method === "PUT") {
      const item = ADMIN_CONTENTS.find(
        (c) => c.id === updateContentMatch[1] && c.deleted_at == null,
      );
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
      item.updated_at = MOCK_NOW;

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
      send(res, 200, stripDeletedField(item));
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
      const now = MOCK_NOW;
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
      const item = ADMIN_CONTENTS.find((c) => c.id === archiveMatch[1] && c.deleted_at == null);
      if (!item) {
        send(res, 404, { error: "not found" });
        return;
      }
      item.archived_at = MOCK_NOW;
      send(res, 200, stripDeletedField(item));
      return;
    }

    // POST /api/v1/admin/contents/:id/unarchive
    const unarchiveMatch = path.match(/^\/api\/v1\/admin\/contents\/([^/]+)\/unarchive$/);
    if (unarchiveMatch && method === "POST") {
      const item = ADMIN_CONTENTS.find((c) => c.id === unarchiveMatch[1] && c.deleted_at == null);
      if (!item) {
        send(res, 404, { error: "not found" });
        return;
      }
      item.archived_at = null;
      send(res, 200, stripDeletedField(item));
      return;
    }

    // GET /api/v1/admin/contents/:contentId/subtitles
    const subtitleTracksMatch = path.match(/^\/api\/v1\/admin\/contents\/([^/]+)\/subtitles$/);
    if (subtitleTracksMatch && method === "GET") {
      if (isContentDeleted(subtitleTracksMatch[1])) {
        send(res, 200, []);
        return;
      }
      send(res, 200, SUBTITLE_TRACKS[subtitleTracksMatch[1]] ?? []);
      return;
    }

    // PUT /api/v1/admin/subtitles/:trackId/status
    const trackStatusMatch = path.match(/^\/api\/v1\/admin\/subtitles\/([^/]+)\/status$/);
    if (trackStatusMatch && method === "PUT") {
      const track = findActiveTrack(trackStatusMatch[1]);
      if (!track) {
        send(res, 404, { error: "not found" });
        return;
      }
      const body = await readJsonBody(req);
      track.status = body.status;
      track.updated_at = MOCK_NOW;
      send(res, 200, track);
      return;
    }

    // GET /api/v1/admin/subtitles/:trackId/cues
    const cuesMatch = path.match(/^\/api\/v1\/admin\/subtitles\/([^/]+)\/cues$/);
    if (cuesMatch && method === "GET") {
      const track = findActiveTrack(cuesMatch[1]);
      if (!track) {
        send(res, 200, []);
        return;
      }
      send(res, 200, SUBTITLE_CUES[track.id] ?? []);
      return;
    }

    // POST /api/v1/admin/subtitles/:trackId/cues
    if (cuesMatch && method === "POST") {
      const track = findActiveTrack(cuesMatch[1]);
      if (!track) {
        send(res, 404, { error: "not found" });
        return;
      }
      const body = await readJsonBody(req);
      const cue = {
        id: `cue-${Date.now()}`,
        track_id: track.id,
        seq: body.seq,
        start_ms: body.start_ms,
        end_ms: body.end_ms,
        text: body.text,
        original_text: body.text,
        flagged: false,
        updated_at: MOCK_NOW,
      };
      SUBTITLE_CUES[track.id] = [...(SUBTITLE_CUES[track.id] ?? []), cue];
      send(res, 201, cue);
      return;
    }

    // PUT /api/v1/admin/subtitles/cues/:cueId
    const cueUpdateMatch = path.match(/^\/api\/v1\/admin\/subtitles\/cues\/([^/]+)$/);
    if (cueUpdateMatch && method === "PUT") {
      const cue = Object.values(SUBTITLE_CUES)
        .flat()
        .find((c) => c.id === cueUpdateMatch[1]);
      if (!cue || !findActiveTrack(cue.track_id)) {
        send(res, 404, { error: "not found" });
        return;
      }
      const body = await readJsonBody(req);
      Object.assign(cue, body, { updated_at: MOCK_NOW });
      send(res, 200, cue);
      return;
    }

    // DELETE /api/v1/admin/subtitles/cues/:cueId
    if (cueUpdateMatch && method === "DELETE") {
      const cueId = cueUpdateMatch[1];
      let found = false;
      for (const trackId of Object.keys(SUBTITLE_CUES)) {
        const idx = SUBTITLE_CUES[trackId].findIndex((c) => c.id === cueId);
        if (idx !== -1) {
          if (!findActiveTrack(trackId)) {
            send(res, 404, { error: "not found" });
            return;
          }
          SUBTITLE_CUES[trackId] = SUBTITLE_CUES[trackId].filter((c) => c.id !== cueId);
          found = true;
        }
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
          !c.deleted_at &&
          (c.title.toLowerCase().includes(q) ||
            c.description.toLowerCase().includes(q) ||
            c.tags.some((t) => t.toLowerCase().includes(q))),
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
}

const isMainModule =
  process.argv[1] != null && import.meta.url === pathToFileURL(realpathSync(process.argv[1])).href;

if (isMainModule) {
  const server = createMockServer();
  const PORT = process.env.MOCK_PORT ?? 3001;
  const HOST = process.env.MOCK_HOST ?? "127.0.0.1";
  server.listen(PORT, HOST, () => {
    console.log(`Mock API server running on http://${HOST}:${PORT}`);
  });
}
