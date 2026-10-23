import { HttpResponse, http } from "msw";
import {
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

// Mutable copies so stories can exercise create/update/delete flows without
// polluting the shared fixtures.
const adminContents: Content[] = ADMIN_CONTENTS.map((c) => ({ ...c }));
const subtitleTracks: Record<string, SubtitleTrack[]> = structuredClone(SUBTITLE_TRACKS);
const subtitleCues: Record<string, SubtitleCue[]> = structuredClone(SUBTITLE_CUES);

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
    let items = CONTENTS;
    if (contentType) items = items.filter((c) => c.content_type === contentType);
    if (channelId) items = items.filter((c) => c.channel_id === channelId);
    return HttpResponse.json(items.slice(0, limit));
  }),

  http.get("/api/v1/contents/:shortId", ({ params }) => {
    const item = CONTENTS.find((c) => c.short_id === params.shortId);
    if (!item) return HttpResponse.json({ error: "not found" }, { status: 404 });
    return HttpResponse.json(item);
  }),

  http.get("/api/v1/contents/:shortId/variants", ({ params }) => {
    const variants = VARIANTS[params.shortId as string] ?? [];
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
        c.title.toLowerCase().includes(q) ||
        c.description.toLowerCase().includes(q) ||
        c.tags.some((t) => t.toLowerCase().includes(q)),
    );
    return HttpResponse.json(results);
  }),

  http.get("/api/v1/admin/contents", ({ request }) => {
    const url = new URL(request.url);
    const status = url.searchParams.get("status");
    const limit = Number(url.searchParams.get("limit") ?? "20");
    const offset = Number(url.searchParams.get("offset") ?? "0");
    let items = adminContents;
    if (status) items = items.filter((c) => c.status === status);
    return HttpResponse.json({
      contents: items.slice(offset, offset + limit),
      total: items.length,
      limit,
      offset,
    });
  }),

  http.put("/api/v1/admin/contents/:id", async ({ params, request }) => {
    const body = (await request.json()) as UpdateContentRequest;
    const idx = adminContents.findIndex((c) => c.id === params.id);
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
    return HttpResponse.json(adminContents[idx]);
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
    const idx = adminContents.findIndex((c) => c.id === params.id);
    if (idx === -1) return HttpResponse.json({ error: "not found" }, { status: 404 });
    adminContents[idx] = { ...adminContents[idx], archived_at: new Date().toISOString() };
    return HttpResponse.json(adminContents[idx]);
  }),

  http.post("/api/v1/admin/contents/:id/unarchive", ({ params }) => {
    const idx = adminContents.findIndex((c) => c.id === params.id);
    if (idx === -1) return HttpResponse.json({ error: "not found" }, { status: 404 });
    adminContents[idx] = { ...adminContents[idx], archived_at: null };
    return HttpResponse.json(adminContents[idx]);
  }),

  http.get("/api/v1/admin/contents/:contentId/subtitles", ({ params }) => {
    const tracks = subtitleTracks[params.contentId as string] ?? [];
    return HttpResponse.json(tracks);
  }),

  http.put("/api/v1/admin/subtitles/:trackId/status", async ({ params, request }) => {
    const body = (await request.json()) as { status: SubtitleTrack["status"] };
    for (const tracks of Object.values(subtitleTracks)) {
      const track = tracks.find((t) => t.id === params.trackId);
      if (track) {
        track.status = body.status;
        track.updated_at = new Date().toISOString();
        return HttpResponse.json(track);
      }
    }
    return HttpResponse.json({ error: "not found" }, { status: 404 });
  }),

  http.get("/api/v1/admin/subtitles/:trackId/cues", ({ params }) => {
    const cues = subtitleCues[params.trackId as string] ?? [];
    return HttpResponse.json(cues);
  }),

  http.post("/api/v1/admin/subtitles/:trackId/cues", async ({ params, request }) => {
    const body = (await request.json()) as {
      seq: number;
      start_ms: number;
      end_ms: number;
      text: string;
    };
    const trackId = params.trackId as string;
    const cue: SubtitleCue = {
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
    subtitleCues[trackId] = [...(subtitleCues[trackId] ?? []), cue];
    return HttpResponse.json(cue);
  }),

  http.put("/api/v1/admin/subtitles/cues/:cueId", async ({ params, request }) => {
    const body = (await request.json()) as { text?: string; start_ms?: number; end_ms?: number };
    for (const cues of Object.values(subtitleCues)) {
      const cue = cues.find((c) => c.id === params.cueId);
      if (cue) {
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
        subtitleCues[trackId] = cues.filter((c) => c.id !== params.cueId);
        return new HttpResponse(null, { status: 204 });
      }
    }
    return HttpResponse.json({ error: "not found" }, { status: 404 });
  }),
];
