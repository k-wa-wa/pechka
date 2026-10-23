import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import VideoPlayer from "./VideoPlayer";

describe("VideoPlayer", () => {
  it("renders video element", () => {
    const { container } = render(
      <LanguageProvider>
        <VideoPlayer variants={[]} shortId="s1" hasSubtitles={false} />
      </LanguageProvider>,
    );

    expect(container.querySelector("video")).toBeDefined();
  });
});
