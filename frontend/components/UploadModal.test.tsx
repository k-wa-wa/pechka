import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import UploadModal from "./UploadModal";

describe("UploadModal", () => {
  it("renders upload form with accessible dialog name", () => {
    render(
      <LanguageProvider>
        <UploadModal channels={[]} onClose={vi.fn()} onUploaded={vi.fn()} />
      </LanguageProvider>,
    );

    const dialog = screen.getByRole("dialog", { name: "Upload Video" });
    expect(dialog.getAttribute("aria-modal")).toBe("true");
    expect(screen.getByText("Upload Video")).toBeDefined();
  });

  it("moves focus into dialog, traps Tab within modal, and handles Escape", () => {
    const onClose = vi.fn();
    render(
      <LanguageProvider>
        <button type="button">Outside Control</button>
        <UploadModal channels={[]} onClose={onClose} onUploaded={vi.fn()} />
      </LanguageProvider>,
    );

    const outside = screen.getByRole("button", { name: "Outside Control" });
    const dialog = screen.getByRole("dialog", { name: "Upload Video" });
    const closeBtn = screen.getByRole("button", { name: "Close" });
    const cancelBtn = screen.getByRole("button", { name: "Cancel" });

    // Upload button is disabled until file + title are filled, so last focusable is Cancel
    expect(dialog.contains(document.activeElement)).toBe(true);
    expect(document.activeElement).toBe(closeBtn);

    fireEvent.keyDown(document, { key: "Tab", shiftKey: true });
    expect(document.activeElement).toBe(cancelBtn);
    expect(document.activeElement).not.toBe(outside);

    fireEvent.keyDown(document, { key: "Tab" });
    expect(document.activeElement).toBe(closeBtn);

    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("restores focus to previously focused trigger element when unmounted", () => {
    const trigger = document.createElement("button");
    trigger.textContent = "Upload Trigger";
    document.body.appendChild(trigger);
    trigger.focus();

    const { unmount } = render(
      <LanguageProvider>
        <UploadModal channels={[]} onClose={vi.fn()} onUploaded={vi.fn()} />
      </LanguageProvider>,
    );

    expect(document.activeElement).not.toBe(trigger);
    unmount();
    expect(document.activeElement).toBe(trigger);
    trigger.remove();
  });
});
