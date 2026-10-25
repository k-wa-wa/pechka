import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import type { MongoContent } from "@/lib/types";
import { CONTENTS } from "@/mocks/fixtures";
import ContentDetailView from "./ContentDetailView";

const VIDEO = CONTENTS.find((c) => c.short_id === "vid001")!;
// Use the PDF variant so DocumentViewer renders an iframe instead of fetching markdown.
const DOCUMENT: MongoContent = {
  ...CONTENTS.find((c) => c.short_id === "doc001")!,
  document_key: "resources/documents/doc001/document.pdf",
  document_mime_type: "application/pdf",
};

function renderView(content: MongoContent) {
  return render(
    <LanguageProvider>
      <ContentDetailView content={content} variants={[]} />
    </LanguageProvider>,
  );
}

describe("ContentDetailView", () => {
  it("renders content detail title", () => {
    renderView(CONTENTS[0]);

    expect(screen.getByText(CONTENTS[0].title)).toBeDefined();
  });

  it.each([
    ["video", VIDEO],
    ["document", DOCUMENT],
  ])("renders the channel as a link with an icon and chevron (%s)", (_, content) => {
    renderView(content);

    const link = screen.getByRole("link", { name: content.channel_name });
    expect(link.getAttribute("href")).toBe(`/channels/${content.channel_id}`);
    const icon = link.querySelector("[data-testid='channel-icon']");
    expect(icon).not.toBeNull();
    expect(icon?.getAttribute("aria-hidden")).toBe("true");
    expect(link.textContent).toContain("›");
  });

  it("falls back to the channel id when the channel name is missing", () => {
    renderView({ ...VIDEO, channel_name: undefined });

    const link = screen.getByRole("link", { name: VIDEO.channel_id });
    expect(link.getAttribute("href")).toBe(`/channels/${VIDEO.channel_id}`);
  });

  it.each([
    ["video", VIDEO],
    ["document", DOCUMENT],
  ])("renders tags as plain #-prefixed text below the title (%s)", (_, content) => {
    renderView(content);

    const title = screen.getByRole("heading", { name: content.title });
    for (const tag of content.tags) {
      const el = screen.getByText(`#${tag}`);
      expect(el.closest("a")).toBeNull();
      expect(title.compareDocumentPosition(el) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    }
  });

  it("renders tags after the description", () => {
    renderView(VIDEO);

    const description = screen.getByText(VIDEO.description);
    const firstTag = screen.getByText(`#${VIDEO.tags[0]}`);
    expect(
      description.compareDocumentPosition(firstTag) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });

  it("renders no tag text when the content has no tags", () => {
    renderView({ ...VIDEO, tags: [] });

    expect(screen.queryByText(/^#/)).toBeNull();
  });
});
