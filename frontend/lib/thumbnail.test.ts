import { describe, expect, it } from "vitest";
import { getPlaceholderUrl, getThumbnailUrl } from "./thumbnail";

describe("getThumbnailUrl", () => {
  it("returns null for null, undefined, or empty string", () => {
    expect(getThumbnailUrl(null)).toBeNull();
    expect(getThumbnailUrl(undefined)).toBeNull();
    expect(getThumbnailUrl("")).toBeNull();
  });

  it("normalizes thumbnail_key without thumbnails/ prefix", () => {
    expect(getThumbnailUrl("2078322465217449984/thumb_01.jpg")).toBe(
      "/thumbnails/2078322465217449984/thumb_01.jpg",
    );
  });

  it("normalizes thumbnail_key with single thumbnails/ prefix", () => {
    expect(getThumbnailUrl("thumbnails/tech-feed-123/thumb_01.jpg")).toBe(
      "/thumbnails/tech-feed-123/thumb_01.jpg",
    );
  });

  it("normalizes thumbnail_key with duplicate thumbnails/ prefix", () => {
    expect(getThumbnailUrl("thumbnails/thumbnails/sid/thumb_01.jpg")).toBe(
      "/thumbnails/sid/thumb_01.jpg",
    );
  });

  it("normalizes thumbnail_key with leading slash", () => {
    expect(getThumbnailUrl("/thumbnails/sid/thumb_01.jpg")).toBe("/thumbnails/sid/thumb_01.jpg");
  });

  it("returns null when thumbnail_key consists only of thumbnails prefix", () => {
    expect(getThumbnailUrl("thumbnails/")).toBeNull();
    expect(getThumbnailUrl("/thumbnails/")).toBeNull();
  });
});

describe("getPlaceholderUrl", () => {
  it("returns placeholder url based on content_type", () => {
    expect(getPlaceholderUrl("video")).toBe("/images/placeholder-video.svg");
    expect(getPlaceholderUrl("image_gallery")).toBe("/images/placeholder-image_gallery.svg");
    expect(getPlaceholderUrl("vr360")).toBe("/images/placeholder-vr360.svg");
  });

  it("falls back to video when content_type is empty or null", () => {
    expect(getPlaceholderUrl(null)).toBe("/images/placeholder-video.svg");
    expect(getPlaceholderUrl(undefined)).toBe("/images/placeholder-video.svg");
    expect(getPlaceholderUrl("")).toBe("/images/placeholder-video.svg");
  });
});
