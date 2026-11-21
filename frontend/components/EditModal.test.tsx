import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import { ADMIN_CONTENTS } from "@/mocks/fixtures";
import EditModal from "./EditModal";

describe("EditModal", () => {
  it("renders edit modal form with accessible dialog name", () => {
    render(
      <LanguageProvider>
        <EditModal content={ADMIN_CONTENTS[0]} onClose={vi.fn()} onSave={vi.fn()} />
      </LanguageProvider>,
    );

    const dialog = screen.getByRole("dialog", { name: "Edit Content" });
    expect(dialog.getAttribute("aria-modal")).toBe("true");
    expect(screen.getByDisplayValue(ADMIN_CONTENTS[0].title)).toBeDefined();
  });

  it("moves focus into dialog, traps Tab navigation, and restores focus on close", () => {
    const onClose = vi.fn();
    render(
      <LanguageProvider>
        <button type="button">Outside Sort</button>
        <EditModal content={ADMIN_CONTENTS[0]} onClose={onClose} onSave={vi.fn()} />
      </LanguageProvider>,
    );

    const outside = screen.getByRole("button", { name: "Outside Sort" });
    const dialog = screen.getByRole("dialog", { name: "Edit Content" });
    const closeBtn = screen.getByRole("button", { name: "Close" });
    const saveBtn = screen.getByRole("button", { name: "Save" });

    expect(dialog.contains(document.activeElement)).toBe(true);
    expect(document.activeElement).toBe(closeBtn);

    fireEvent.keyDown(document, { key: "Tab", shiftKey: true });
    expect(document.activeElement).toBe(saveBtn);
    expect(document.activeElement).not.toBe(outside);

    fireEvent.keyDown(document, { key: "Tab" });
    expect(document.activeElement).toBe(closeBtn);

    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("restores focus to previously focused trigger element when unmounted", () => {
    const trigger = document.createElement("button");
    trigger.textContent = "Edit Trigger";
    document.body.appendChild(trigger);
    trigger.focus();

    const { unmount } = render(
      <LanguageProvider>
        <EditModal content={ADMIN_CONTENTS[0]} onClose={vi.fn()} onSave={vi.fn()} />
      </LanguageProvider>,
    );

    expect(document.activeElement).not.toBe(trigger);
    unmount();
    expect(document.activeElement).toBe(trigger);
    trigger.remove();
  });
});
