import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import SettingsModal from "./SettingsModal";

describe("SettingsModal", () => {
  it("renders modal when open", () => {
    render(
      <LanguageProvider>
        <SettingsModal isOpen={true} onClose={vi.fn()} />
      </LanguageProvider>,
    );

    expect(screen.getByText("Settings")).toBeDefined();
  });
});
