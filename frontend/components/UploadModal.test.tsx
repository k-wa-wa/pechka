import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import UploadModal from "./UploadModal";

describe("UploadModal", () => {
  it("renders upload form when open", () => {
    render(
      <LanguageProvider>
        <UploadModal channels={[]} onClose={vi.fn()} onUploaded={vi.fn()} />
      </LanguageProvider>,
    );

    expect(screen.getByText("Upload Video")).toBeDefined();
  });
});
