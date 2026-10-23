import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  archiveContent,
  deleteSubtitleCue,
  getAdminContents,
  getChannel,
  getChannels,
  getContent,
  getContents,
  getSubtitleCues,
  getSubtitleTracks,
  getVariants,
  insertSubtitleCue,
  searchContents,
  unarchiveContent,
  updateContent,
  updateSubtitleCue,
  updateSubtitleTrackStatus,
} from "./api";

describe("lib/api", () => {
  const originalFetch = global.fetch;

  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    global.fetch = originalFetch;
  });

  it("getContents fetches contents list with query params", async () => {
    const mockData = [{ id: "1", short_id: "s1", title: "Test" }];
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => mockData,
    });

    const res = await getContents({
      content_type: "video",
      limit: 10,
      offset: 0,
      channel_id: "ch1",
    });
    expect(res).toEqual(mockData);
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining(
        "/api/v1/contents?limit=10&offset=0&content_type=video&channel_id=ch1",
      ),
      expect.any(Object),
    );
  });

  it("getContent fetches single content by shortId", async () => {
    const mockData = { id: "1", short_id: "s1", title: "Test" };
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => mockData,
    });

    const res = await getContent("s1");
    expect(res).toEqual(mockData);
  });

  it("getContent throws on API error", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      statusText: "Not Found",
    });

    await expect(getContent("unknown")).rejects.toThrow("API error 404: Not Found");
  });

  it("getVariants fetches variants", async () => {
    const mockData = [{ variant_type: "1080p" }];
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => mockData,
    });

    const res = await getVariants("s1");
    expect(res).toEqual(mockData);
  });

  it("searchContents queries search endpoint", async () => {
    const mockData = [{ short_id: "s1", title: "Found" }];
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => mockData,
    });

    const res = await searchContents("test", { limit: 5, offset: 0 });
    expect(res).toEqual(mockData);
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/search?q=test&limit=5&offset=0"),
      undefined,
    );
  });

  it("getAdminContents parses response", async () => {
    const mockData = { contents: [{ id: "1", short_id: "s1" }], total: 100 };
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => mockData,
    });

    const res = await getAdminContents({ limit: 10, offset: 0, status: "ready" });
    expect(res.contents).toHaveLength(1);
    expect(res.total).toBe(100);
  });

  it("updateContent sends PUT request", async () => {
    const mockData = { id: "1", short_id: "s1", title: "Updated" };
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => mockData,
    });

    const res = await updateContent("1", { title: "Updated" });
    expect(res).toEqual(mockData);
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/admin/contents/1"),
      expect.objectContaining({ method: "PUT" }),
    );
  });

  it("archiveContent and unarchiveContent work", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ id: "1" }),
    });
    await expect(archiveContent("1")).resolves.toEqual({ id: "1" });
    await expect(unarchiveContent("1")).resolves.toEqual({ id: "1" });
  });

  it("channels API operations work as expected", async () => {
    const mockChannels = [{ id: "c1", name: "Channel 1" }];
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => mockChannels,
    });

    const list = await getChannels();
    expect(list).toEqual(mockChannels);

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => mockChannels[0],
    });
    const single = await getChannel("c1");
    expect(single).toEqual(mockChannels[0]);
  });

  it("subtitles API operations work as expected", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [{ id: "t1", language: "ja", status: "draft" }],
    });
    const tracks = await getSubtitleTracks("c1");
    expect(tracks).toHaveLength(1);

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ id: "t1", language: "ja", status: "published" }),
    });
    const updatedTrack = await updateSubtitleTrackStatus("t1", "published");
    expect(updatedTrack.status).toBe("published");

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [{ id: 1, text: "Hello" }],
    });
    const cues = await getSubtitleCues("t1");
    expect(cues).toHaveLength(1);

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ id: 1, text: "Updated" }),
    });
    const updatedCue = await updateSubtitleCue("1", { text: "Updated" });
    expect(updatedCue.text).toBe("Updated");

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ id: 2, text: "New" }),
    });
    const newCue = await insertSubtitleCue("t1", {
      seq: 1,
      start_ms: 0,
      end_ms: 1000,
      text: "New",
    });
    expect(newCue.text).toBe("New");

    global.fetch = vi.fn().mockResolvedValue({ ok: true });
    await expect(deleteSubtitleCue("1")).resolves.toBeUndefined();
  });
});
