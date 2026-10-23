import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import { ADMIN_CONTENTS } from "@/mocks/fixtures";
import SubtitleEditorModal from "./SubtitleEditorModal";

describe("SubtitleEditorModal", () => {
  it("renders modal when open", () => {
    const { container } = render(
      <LanguageProvider>
        <SubtitleEditorModal content={ADMIN_CONTENTS[0]} onClose={vi.fn()} />
      </LanguageProvider>,
    );

    expect(container).toBeDefined();
  });
});
