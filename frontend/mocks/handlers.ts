import { HttpResponse, http } from "msw";
import {
  type AdminContentSort,
  type Content,
  resolveDocumentViewerKind,
  type SubtitleCue,
  type SubtitleTrack,
  type UpdateContentRequest,
} from "@/lib/types";
import {
  ADMIN_CONTENTS,
  CHANNELS,
  CONTENTS,
  SAMPLE_MARKDOWN,
  SEARCH_RESULTS,
  SUBTITLE_CUES,
  SUBTITLE_TRACKS,
  VARIANTS,
} from "./fixtures";

type MockAdminContent = Content & { deleted_at?: string | null };

// Mutable copies so stories can exercise create/update/delete flows without
// polluting the shared fixtures.
const adminContents: MockAdminContent[] = ADMIN_CONTENTS.map((c) => ({ ...c, deleted_at: null }));
const subtitleTracks: Record<string, SubtitleTrack[]> = structuredClone(SUBTITLE_TRACKS);
const subtitleCues: Record<string, SubtitleCue[]> = structuredClone(SUBTITLE_CUES);

const VALID_ADMIN_SORTS = new Set<AdminContentSort>([
  "updated_desc",
  "updated_asc",
  "created_desc",
  "created_asc",
  "title_asc",
  "title_desc",
]);

const VALID_CHANNEL_ID_RE = /^[a-z0-9][a-z0-9-]{0,49}$/;

function parseStrictLimit(rawParam: string | null): number {
  if (rawParam === null || !/^[+-]?\d+$/.test(rawParam)) {
    return 20;
  }
  const parsed = Number(rawParam);
  return Number.isSafeInteger(parsed) && parsed >= 1 && parsed <= 100 ? parsed : 20;
}

function compareCodePoints(a: string, b: string): number {
  if (a === b) return 0;
  const iterA = a[Symbol.iterator]();
  const iterB = b[Symbol.iterator]();
  while (true) {
    const nextA = iterA.next();
    const nextB = iterB.next();
    if (nextA.done && nextB.done) return 0;
    if (nextA.done) return -1;
    if (nextB.done) return 1;
    const cpA = nextA.value.codePointAt(0) ?? 0;
    const cpB = nextB.value.codePointAt(0) ?? 0;
    if (cpA !== cpB) return cpA < cpB ? -1 : 1;
  }
}

function sortMeta(sort: AdminContentSort): {
  field: "updated_at" | "created_at" | "title";
  dir: 1 | -1;
  isTitle: boolean;
} {
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
  }
}

function toBase64Url(str: string): string {
  const bytes = new TextEncoder().encode(str);
  let binary = "";
  for (const b of bytes) binary += String.fromCharCode(b);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function fromBase64Url(token: string): string | null {
  if (!token || !/^[A-Za-z0-9_-]+$/.test(token) || token.length % 4 === 1) return null;
  try {
    const base64 = token.replace(/-/g, "+").replace(/_/g, "/");
    const pad = base64.length % 4 === 0 ? "" : "=".repeat(4 - (base64.length % 4));
    const binary = atob(base64 + pad);
    const bytes = Uint8Array.from(binary, (c) => c.charCodeAt(0));
    return new TextDecoder().decode(bytes);
  } catch {
    return null;
  }
}

const INT64_MAX_DIGITS = "9223372036854775807";
const INT64_MIN_ABS_DIGITS = "9223372036854775808";

function extractRawJsonObjectField(jsonText: string, targetKey: string): string | null {
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
  const readString = (): string | null => {
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
  const skipValue = (): string | null => {
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
  let found: string | null = null;
  while (i < n) {
    skipWs();
    if (jsonText[i] === "}") break;
    const rawKey = readString();
    if (rawKey === null) return null;
    const key = JSON.parse(rawKey) as string;
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

function parseStrictInt64Token(rawToken: string | null): number | null {
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

interface DecodedCursor {
  sort: AdminContentSort;
  v: string | number;
  id: string;
}

function encodeAdminCursor(sort: AdminContentSort, item: MockAdminContent): string {
  const meta = sortMeta(sort);
  const v = meta.isTitle ? item.title : Date.parse(item[meta.field]);
  return toBase64Url(JSON.stringify({ sort, v, id: item.short_id }));
}

function decodeAdminCursor(token: string, expectedSort: AdminContentSort): DecodedCursor | null {
  const raw = fromBase64Url(token);
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as Partial<DecodedCursor>;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return null;
    if (!parsed.sort || !VALID_ADMIN_SORTS.has(parsed.sort) || parsed.sort !== expectedSort)
      return null;
    if (typeof parsed.id !== "string" || parsed.id === "") return null;
    const meta = sortMeta(parsed.sort);
    let v: string | number;
    if (meta.isTitle) {
      if (typeof parsed.v !== "string") return null;
      v = parsed.v;
    } else {
      const int64V = parseStrictInt64Token(extractRawJsonObjectField(raw, "v"));
      if (int64V === null) return null;
      v = int64V;
    }
    return { sort: parsed.sort, v, id: parsed.id };
  } catch {
    return null;
  }
}

function compareAdminItems(
  a: MockAdminContent,
  b: MockAdminContent,
  sort: AdminContentSort,
): number {
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

function compareItemToCursor(
  item: MockAdminContent,
  cursor: DecodedCursor,
  sort: AdminContentSort,
): number {
  const meta = sortMeta(sort);
  let cmp = 0;
  if (meta.isTitle) {
    cmp = compareCodePoints(item.title, String(cursor.v));
  } else {
    const ti = Date.parse(item[meta.field]);
    const tc = Number(cursor.v);
    cmp = ti === tc ? 0 : ti < tc ? -1 : 1;
  }
  if (cmp !== 0) return cmp * meta.dir;
  return compareCodePoints(item.short_id, cursor.id);
}

function stripDeletedField({ deleted_at: _deletedAt, ...rest }: MockAdminContent): Content {
  return rest;
}

function isDeletedShortId(shortId: string): boolean {
  const item = adminContents.find((c) => c.short_id === shortId);
  return item != null && item.deleted_at != null;
}

function isContentDeleted(contentId: string): boolean {
  const item = adminContents.find((c) => c.id === contentId);
  return item != null && item.deleted_at != null;
}

function findActiveTrack(trackId: string): SubtitleTrack | null {
  for (const tracks of Object.values(subtitleTracks)) {
    const track = tracks.find((t) => t.id === trackId);
    if (track) {
      return isContentDeleted(track.content_id) ? null : track;
    }
  }
  return null;
}

export const handlers = [
  http.get("/api/v1/channels", () => {
    return HttpResponse.json(CHANNELS);
  }),

  http.get("/api/v1/channels/:id", ({ params }) => {
    const ch = CHANNELS.find((c) => c.id === params.id);
    if (!ch) return HttpResponse.json({ error: "not found" }, { status: 404 });
    return HttpResponse.json(ch);
  }),

  http.get("/api/v1/contents", ({ request }) => {
    const url = new URL(request.url);
    const contentType = url.searchParams.get("content_type");
    const channelId = url.searchParams.get("channel_id");
    const limit = Number(url.searchParams.get("limit") ?? "100");
    let items = CONTENTS.filter((c) => !isDeletedShortId(c.short_id));
    if (contentType) items = items.filter((c) => c.content_type === contentType);
    if (channelId) items = items.filter((c) => c.channel_id === channelId);
    return HttpResponse.json(items.slice(0, limit));
  }),

  http.get("/api/v1/contents/:shortId", ({ params }) => {
    const shortId = params.shortId as string;
    if (isDeletedShortId(shortId)) {
      return HttpResponse.json({ error: "not found" }, { status: 404 });
    }
    const item = CONTENTS.find((c) => c.short_id === shortId);
    if (!item) return HttpResponse.json({ error: "not found" }, { status: 404 });
    return HttpResponse.json(item);
  }),

  http.get("/api/v1/contents/:shortId/variants", ({ params }) => {
    const shortId = params.shortId as string;
    if (isDeletedShortId(shortId)) {
      return HttpResponse.json({ error: "not found" }, { status: 404 });
    }
    const variants = VARIANTS[shortId] ?? [];
    return HttpResponse.json(variants);
  }),

  http.get("/resources/documents/:shortId/:filename", ({ params }) => {
    const filename = String(params.filename ?? "");
    if (filename.endsWith(".md") || filename.endsWith(".markdown")) {
      return new HttpResponse(SAMPLE_MARKDOWN, {
        status: 200,
        headers: { "Content-Type": "text/markdown; charset=utf-8" },
      });
    }
    if (filename.endsWith(".pdf")) {
      return new HttpResponse("%PDF-1.7\n%mock pdf\n%%EOF\n", {
        status: 200,
        headers: { "Content-Type": "application/pdf" },
      });
    }
    return HttpResponse.json({ error: "not found" }, { status: 404 });
  }),

  http.get("/api/v1/search", ({ request }) => {
    const url = new URL(request.url);
    const q = (url.searchParams.get("q") ?? "").toLowerCase();
    const results = SEARCH_RESULTS.filter(
      (c) =>
        !isDeletedShortId(c.short_id) &&
        (c.title.toLowerCase().includes(q) ||
          c.description.toLowerCase().includes(q) ||
          c.tags.some((t) => t.toLowerCase().includes(q))),
    );
    return HttpResponse.json(results);
  }),

  http.get("/api/v1/admin/contents", ({ request }) => {
    const url = new URL(request.url);
    const status = url.searchParams.get("status");
    const channelId = url.searchParams.get("channel_id");
    if (channelId && !VALID_CHANNEL_ID_RE.test(channelId)) {
      return HttpResponse.json({ message: "invalid channel_id" }, { status: 400 });
    }
    const limit = parseStrictLimit(url.searchParams.get("limit"));
    const sortParam = (url.searchParams.get("sort") || "updated_desc") as AdminContentSort;
    if (!VALID_ADMIN_SORTS.has(sortParam)) {
      return HttpResponse.json({ message: "invalid sort" }, { status: 400 });
    }
    const afterParam = url.searchParams.get("after") ?? "";
    const beforeParam = url.searchParams.get("before") ?? "";
    if (afterParam && beforeParam) {
      return HttpResponse.json(
        { message: "after and before cannot be specified together" },
        { status: 400 },
      );
    }
    const afterCur = afterParam ? decodeAdminCursor(afterParam, sortParam) : null;
    const beforeCur = beforeParam ? decodeAdminCursor(beforeParam, sortParam) : null;
    if ((afterParam && !afterCur) || (beforeParam && !beforeCur)) {
      return HttpResponse.json({ message: "invalid cursor" }, { status: 400 });
    }

    let items = adminContents.filter((c) => c.deleted_at == null);
    if (status) items = items.filter((c) => c.status === status);
    if (channelId) items = items.filter((c) => c.channel_id === channelId);

    const sorted = [...items].sort((a, b) => compareAdminItems(a, b, sortParam));
    const total = sorted.length;

    let pageItems: MockAdminContent[] = [];
    let hasNext = false;
    let hasPrev = false;
    const meta = sortMeta(sortParam);

    if (afterCur) {
      const afterMatches = sorted.filter((c) => compareItemToCursor(c, afterCur, sortParam) > 0);
      pageItems = afterMatches.slice(0, limit);
      if (pageItems.length > 0) {
        hasNext = afterMatches.length > limit;
        const firstCur: DecodedCursor = {
          sort: sortParam,
          v: meta.isTitle ? pageItems[0].title : Date.parse(pageItems[0][meta.field]),
          id: pageItems[0].short_id,
        };
        hasPrev = sorted.some((c) => compareItemToCursor(c, firstCur, sortParam) < 0);
      }
    } else if (beforeCur) {
      const beforeMatches = sorted.filter((c) => compareItemToCursor(c, beforeCur, sortParam) < 0);
      hasPrev = beforeMatches.length > limit;
      pageItems = beforeMatches.slice(-limit);
      if (pageItems.length > 0) {
        const last = pageItems[pageItems.length - 1];
        const lastCur: DecodedCursor = {
          sort: sortParam,
          v: meta.isTitle ? last.title : Date.parse(last[meta.field]),
          id: last.short_id,
        };
        hasNext = sorted.some((c) => compareItemToCursor(c, lastCur, sortParam) > 0);
      }
    } else {
      pageItems = sorted.slice(0, limit);
      hasNext = sorted.length > limit;
      hasPrev = false;
    }

    const nextCursor =
      hasNext && pageItems.length > 0
        ? encodeAdminCursor(sortParam, pageItems[pageItems.length - 1])
        : null;
    const prevCursor =
      hasPrev && pageItems.length > 0 ? encodeAdminCursor(sortParam, pageItems[0]) : null;

    return HttpResponse.json({
      contents: pageItems.map(({ deleted_at: _deletedAt, ...rest }) => rest),
      total,
      limit,
      next_cursor: nextCursor,
      prev_cursor: prevCursor,
      has_next: hasNext,
      has_prev: hasPrev,
    });
  }),

  http.delete("/api/v1/admin/contents/:id", ({ params }) => {
    const idx = adminContents.findIndex((c) => c.id === params.id);
    if (idx !== -1 && adminContents[idx].deleted_at == null) {
      const now = new Date().toISOString();
      adminContents[idx] = { ...adminContents[idx], deleted_at: now, updated_at: now };
    }
    return new HttpResponse(null, { status: 204 });
  }),

  http.put("/api/v1/admin/contents/:id", async ({ params, request }) => {
    const body = (await request.json()) as UpdateContentRequest;
    const idx = adminContents.findIndex((c) => c.id === params.id && c.deleted_at == null);
    if (idx === -1) return HttpResponse.json({ error: "not found" }, { status: 404 });
    adminContents[idx] = {
      ...adminContents[idx],
      ...(body.channel_id != null && { channel_id: body.channel_id }),
      ...(body.title != null && { title: body.title }),
      ...(body.description != null && { description: body.description }),
      ...(body.tags != null && { tags: body.tags }),
      ...(body.status != null && { status: body.status }),
      updated_at: new Date().toISOString(),
    };
    return HttpResponse.json(stripDeletedField(adminContents[idx]));
  }),

  http.post("/api/v1/admin/contents/upload", async ({ request }) => {
    const formData = await request.formData();
    const title = formData.get("title");
    const file = formData.get("file");
    if (!file || typeof title !== "string" || title.trim() === "") {
      return HttpResponse.json({ error: "file and title are required" }, { status: 400 });
    }
    const description = formData.get("description");
    const tagsParam = formData.get("tags");
    const tags =
      typeof tagsParam === "string"
        ? tagsParam
            .split(",")
            .map((t) => t.trim())
            .filter((t) => t !== "")
        : [];
    const channelId = (request.headers.get("X-Channel-Id") ?? "").trim();
    if (!channelId) {
      return HttpResponse.json({ error: "channel_id is required" }, { status: 400 });
    }
    const now = new Date().toISOString();
    const shortId = `up${Date.now()}`;
    const uploadedFilename = file instanceof File ? file.name : "";
    const docKind = resolveDocumentViewerKind(null, uploadedFilename);
    const isDoc = docKind != null;
    const docKey =
      docKind === "markdown"
        ? `resources/documents/${shortId}/document.md`
        : docKind === "pdf"
          ? `resources/documents/${shortId}/document.pdf`
          : null;
    const docMime =
      docKind === "markdown" ? "text/markdown" : docKind === "pdf" ? "application/pdf" : null;

    const content: Content = {
      id: `upload-${Date.now()}`,
      short_id: shortId,
      content_type: isDoc ? "document" : "video",
      channel_id: channelId,
      disc_id: null,
      title,
      description: typeof description === "string" ? description : "",
      duration_seconds: null,
      is_360: false,
      tags,
      status: isDoc ? "ready" : "pending",
      document_key: docKey,
      document_mime_type: docMime,
      published_at: isDoc ? now : null,
      archived_at: null,
      created_at: now,
      updated_at: now,
    };
    adminContents.unshift(content);
    return HttpResponse.json(
      { content, workflow_name: isDoc ? "" : "video-upload-mock" },
      { status: 201 },
    );
  }),

  http.post("/api/v1/admin/contents/:id/archive", ({ params }) => {
    const idx = adminContents.findIndex((c) => c.id === params.id && c.deleted_at == null);
    if (idx === -1) return HttpResponse.json({ error: "not found" }, { status: 404 });
    adminContents[idx] = { ...adminContents[idx], archived_at: new Date().toISOString() };
    return HttpResponse.json(stripDeletedField(adminContents[idx]));
  }),

  http.post("/api/v1/admin/contents/:id/unarchive", ({ params }) => {
    const idx = adminContents.findIndex((c) => c.id === params.id && c.deleted_at == null);
    if (idx === -1) return HttpResponse.json({ error: "not found" }, { status: 404 });
    adminContents[idx] = { ...adminContents[idx], archived_at: null };
    return HttpResponse.json(stripDeletedField(adminContents[idx]));
  }),

  http.get("/api/v1/admin/contents/:contentId/subtitles", ({ params }) => {
    const contentId = params.contentId as string;
    if (isContentDeleted(contentId)) {
      return HttpResponse.json([]);
    }
    const tracks = subtitleTracks[contentId] ?? [];
    return HttpResponse.json(tracks);
  }),

  http.put("/api/v1/admin/subtitles/:trackId/status", async ({ params, request }) => {
    const body = (await request.json()) as { status: SubtitleTrack["status"] };
    const track = findActiveTrack(params.trackId as string);
    if (!track) {
      return HttpResponse.json({ error: "not found" }, { status: 404 });
    }
    track.status = body.status;
    track.updated_at = new Date().toISOString();
    return HttpResponse.json(track);
  }),

  http.get("/api/v1/admin/subtitles/:trackId/cues", ({ params }) => {
    const track = findActiveTrack(params.trackId as string);
    if (!track) {
      return HttpResponse.json([]);
    }
    const cues = subtitleCues[track.id] ?? [];
    return HttpResponse.json(cues);
  }),

  http.post("/api/v1/admin/subtitles/:trackId/cues", async ({ params, request }) => {
    const track = findActiveTrack(params.trackId as string);
    if (!track) {
      return HttpResponse.json({ error: "not found" }, { status: 404 });
    }
    const body = (await request.json()) as {
      seq: number;
      start_ms: number;
      end_ms: number;
      text: string;
    };
    const cue: SubtitleCue = {
      id: `cue-${Date.now()}`,
      track_id: track.id,
      seq: body.seq,
      start_ms: body.start_ms,
      end_ms: body.end_ms,
      text: body.text,
      original_text: body.text,
      flagged: false,
      updated_at: new Date().toISOString(),
    };
    subtitleCues[track.id] = [...(subtitleCues[track.id] ?? []), cue];
    return HttpResponse.json(cue, { status: 201 });
  }),

  http.put("/api/v1/admin/subtitles/cues/:cueId", async ({ params, request }) => {
    const body = (await request.json()) as { text?: string; start_ms?: number; end_ms?: number };
    for (const cues of Object.values(subtitleCues)) {
      const cue = cues.find((c) => c.id === params.cueId);
      if (cue) {
        if (!findActiveTrack(cue.track_id)) {
          return HttpResponse.json({ error: "not found" }, { status: 404 });
        }
        Object.assign(cue, body, { updated_at: new Date().toISOString() });
        return HttpResponse.json(cue);
      }
    }
    return HttpResponse.json({ error: "not found" }, { status: 404 });
  }),

  http.delete("/api/v1/admin/subtitles/cues/:cueId", ({ params }) => {
    for (const [trackId, cues] of Object.entries(subtitleCues)) {
      const idx = cues.findIndex((c) => c.id === params.cueId);
      if (idx !== -1) {
        if (!findActiveTrack(trackId)) {
          return HttpResponse.json({ error: "not found" }, { status: 404 });
        }
        subtitleCues[trackId] = cues.filter((c) => c.id !== params.cueId);
        return new HttpResponse(null, { status: 204 });
      }
    }
    return HttpResponse.json({ error: "not found" }, { status: 404 });
  }),
];
