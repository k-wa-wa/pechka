import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import ContentPlayer from "./ContentPlayer";

describe("ContentPlayer", () => {
  it("renders content player", () => {
    const { container } = render(
      <LanguageProvider>
        <ContentPlayer variants={[]} isVr={false} shortId="s1" hasSubtitles={false} />
      </LanguageProvider>,
    );

    expect(container).toBeDefined();
  });
});
