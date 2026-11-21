import { fireEvent, render, screen } from "@testing-library/react";
import { createElement, useRef } from "react";
import { describe, expect, it, vi } from "vitest";
import { useDialogA11y } from "./useDialogA11y";

function TestDialog({
  onClose,
  closeDisabled = false,
}: {
  onClose: () => void;
  closeDisabled?: boolean;
}) {
  const dialogRef = useRef<HTMLDivElement>(null);
  useDialogA11y({ dialogRef, onClose, closeDisabled });

  return createElement(
    "div",
    { ref: dialogRef, role: "dialog", "aria-modal": "true", tabIndex: -1 },
    createElement("button", { type: "button" }, "First"),
    createElement("input", { "aria-label": "Middle" }),
    createElement("button", { type: "button", disabled: true }, "Disabled"),
    createElement("button", { type: "button" }, "Last"),
  );
}

describe("useDialogA11y", () => {
  it("moves focus inside dialog, traps Tab/Shift+Tab, handles Escape, and restores focus", () => {
    const onClose = vi.fn();
    const trigger = document.createElement("button");
    trigger.textContent = "Open";
    document.body.appendChild(trigger);
    trigger.focus();

    const { unmount } = render(createElement(TestDialog, { onClose }));
    const first = screen.getByText("First");
    const last = screen.getByText("Last");
    expect(document.activeElement).toBe(first);

    // Shift+Tab from first wraps to last (skipping disabled button)
    fireEvent.keyDown(document, { key: "Tab", shiftKey: true });
    expect(document.activeElement).toBe(last);

    // Tab from last wraps back to first
    fireEvent.keyDown(document, { key: "Tab" });
    expect(document.activeElement).toBe(first);

    // Escape triggers onClose
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);

    // Unmount restores focus to trigger
    unmount();
    expect(document.activeElement).toBe(trigger);
    trigger.remove();
  });

  it("ignores Escape when closeDisabled is true", () => {
    const onClose = vi.fn();
    render(createElement(TestDialog, { onClose, closeDisabled: true }));
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).not.toHaveBeenCalled();
  });
});
