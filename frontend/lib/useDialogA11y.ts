import { type RefObject, useEffect, useRef } from "react";

const FOCUSABLE_SELECTOR = [
  "a[href]",
  "button:not([disabled])",
  "input:not([disabled]):not([type='hidden'])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  "[tabindex]:not([tabindex='-1'])",
].join(", ");

function getFocusableElements(container: HTMLElement): HTMLElement[] {
  return Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter(
    (el) => el.getAttribute("aria-hidden") !== "true",
  );
}

interface UseDialogA11yOptions {
  dialogRef: RefObject<HTMLElement | null>;
  onClose: () => void;
  closeDisabled?: boolean;
}

/**
 * Manages modal dialog focus entry, Tab/Shift+Tab trapping, Escape dismissal,
 * and focus restoration upon unmount.
 */
export function useDialogA11y({
  dialogRef,
  onClose,
  closeDisabled = false,
}: UseDialogA11yOptions): void {
  const onCloseRef = useRef(onClose);
  const closeDisabledRef = useRef(closeDisabled);
  onCloseRef.current = onClose;
  closeDisabledRef.current = closeDisabled;

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    const previousActive =
      document.activeElement instanceof HTMLElement ? document.activeElement : null;

    const focusables = getFocusableElements(dialog);
    const initialTarget = focusables[0] ?? dialog;
    initialTarget.focus();

    function handleKeyDown(e: KeyboardEvent) {
      const currentDialog = dialogRef.current;
      if (!currentDialog) return;

      if (e.key === "Escape") {
        if (!closeDisabledRef.current) {
          e.preventDefault();
          onCloseRef.current();
        }
        return;
      }

      if (e.key !== "Tab") return;

      const items = getFocusableElements(currentDialog);
      if (items.length === 0) {
        e.preventDefault();
        currentDialog.focus();
        return;
      }

      const first = items[0];
      const last = items[items.length - 1];
      const active = document.activeElement;

      if (!currentDialog.contains(active)) {
        e.preventDefault();
        (e.shiftKey ? last : first).focus();
        return;
      }

      if (e.shiftKey) {
        if (active === first || active === currentDialog) {
          e.preventDefault();
          last.focus();
        }
      } else if (active === last) {
        e.preventDefault();
        first.focus();
      }
    }

    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      if (previousActive?.isConnected) {
        previousActive.focus();
      }
    };
  }, [dialogRef]);
}
