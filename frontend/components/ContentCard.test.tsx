import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import { CONTENTS } from "@/mocks/fixtures";
import ContentCard from "./ContentCard";

describe("ContentCard", () => {
  it("renders content card information", () => {
    render(
      <LanguageProvider>
        <ContentCard content={CONTENTS[0]} />
      </LanguageProvider>,
    );

    expect(screen.getByText(CONTENTS[0].title)).toBeDefined();
  });

  it("normalizes thumbnail url when thumbnail_key has thumbnails/ prefix", () => {
    const item = {
      ...CONTENTS[0],
      thumbnail_key: "thumbnails/test-item/thumb_01.jpg",
    };
    render(
      <LanguageProvider>
        <ContentCard content={item} />
      </LanguageProvider>,
    );

    const img = screen.getByAltText(item.title) as HTMLImageElement;
    expect(img.getAttribute("src")).toBe("/thumbnails/test-item/thumb_01.jpg");
    expect(img.style.opacity).toBe("0");

    fireEvent.load(img);
    expect(img.style.opacity).toBe("1");
  });

  it("normalizes thumbnail url when thumbnail_key lacks thumbnails/ prefix", () => {
    const item = {
      ...CONTENTS[0],
      thumbnail_key: "test-item/thumb_01.jpg",
    };
    render(
      <LanguageProvider>
        <ContentCard content={item} />
      </LanguageProvider>,
    );

    const img = screen.getByAltText(item.title) as HTMLImageElement;
    expect(img.getAttribute("src")).toBe("/thumbnails/test-item/thumb_01.jpg");
  });

  it("renders placeholder when thumbnail_key is null", () => {
    const item = {
      ...CONTENTS[0],
      thumbnail_key: null,
    };
    render(
      <LanguageProvider>
        <ContentCard content={item} />
      </LanguageProvider>,
    );

    const img = screen.getByAltText(item.title) as HTMLImageElement;
    expect(img.getAttribute("src")).toBe("/images/placeholder-video.svg");
  });

  it("keeps thumbnail opacity at 0 when image fails to load (onError)", () => {
    const item = {
      ...CONTENTS[0],
      thumbnail_key: "thumbnails/broken-image/thumb_01.jpg",
    };
    render(
      <LanguageProvider>
        <ContentCard content={item} />
      </LanguageProvider>,
    );

    const img = screen.getByAltText(item.title) as HTMLImageElement;
    expect(img.style.opacity).toBe("0");

    fireEvent.error(img);
    expect(img.style.opacity).toBe("0");
  });

  it("shows image immediately on mount if image completed loading before hydration", () => {
    const origComplete = Object.getOwnPropertyDescriptor(HTMLImageElement.prototype, "complete");
    const origNaturalWidth = Object.getOwnPropertyDescriptor(
      HTMLImageElement.prototype,
      "naturalWidth",
    );

    try {
      Object.defineProperty(HTMLImageElement.prototype, "complete", {
        configurable: true,
        get: () => true,
      });
      Object.defineProperty(HTMLImageElement.prototype, "naturalWidth", {
        configurable: true,
        get: () => 320,
      });

      const item = {
        ...CONTENTS[0],
        thumbnail_key: "thumbnails/preloaded/thumb_01.jpg",
      };
      render(
        <LanguageProvider>
          <ContentCard content={item} />
        </LanguageProvider>,
      );

      const img = screen.getByAltText(item.title) as HTMLImageElement;
      expect(img.style.opacity).toBe("1");
    } finally {
      if (origComplete) {
        Object.defineProperty(HTMLImageElement.prototype, "complete", origComplete);
      } else {
        delete (HTMLImageElement.prototype as unknown as { complete?: boolean }).complete;
      }
      if (origNaturalWidth) {
        Object.defineProperty(HTMLImageElement.prototype, "naturalWidth", origNaturalWidth);
      } else {
        delete (HTMLImageElement.prototype as unknown as { naturalWidth?: number }).naturalWidth;
      }
    }
  });
});
