import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import SearchModal from "./SearchModal";

describe("SearchModal", () => {
  it("renders search modal when open", () => {
    render(
      <LanguageProvider>
        <SearchModal isOpen={true} onClose={vi.fn()} />
      </LanguageProvider>,
    );

    expect(screen.getByPlaceholderText("Search contents...")).toBeDefined();
  });
});
