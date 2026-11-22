import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import SearchModal from "./SearchModal";

describe("SearchModal", () => {
  it("renders search modal when open, traps Tab inside dialog, and handles Escape", () => {
    const onClose = vi.fn();
    render(
      <LanguageProvider>
        <button type="button">Background Admin Action</button>
        <SearchModal isOpen={true} onClose={onClose} />
      </LanguageProvider>,
    );

    const input = screen.getByPlaceholderText("Search contents...");
    const dialog = screen.getByRole("dialog", { name: "Search" });
    expect(document.activeElement).toBe(input);

    fireEvent.keyDown(document, { key: "Tab" });
    expect(dialog.contains(document.activeElement)).toBe(true);
    expect(document.activeElement).not.toBe(screen.getByText("Background Admin Action"));

    fireEvent.keyDown(document, { key: "Tab", shiftKey: true });
    expect(dialog.contains(document.activeElement)).toBe(true);

    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
