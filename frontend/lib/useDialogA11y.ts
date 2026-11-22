import { type RefObject, useEffect, useRef } from "react";

const FOCUSABLE_SELECTOR = [
  "a[href]",
  "button:not([disabled])",
  "input:not([disabled]):not([type='hidden'])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  "[tabindex]:not([tabindex='-1'])",
].join(", ");

function getEffectiveZIndex(el: HTMLElement): number {
  let maxZ = 0;
  let cur: HTMLElement | null = el;
  while (cur && cur !== document.body) {
    const raw = cur.style.zIndex || getComputedStyle(cur).zIndex;
    const parsed = Number.parseInt(raw, 10);
    if (Number.isFinite(parsed) && parsed > maxZ) {
      maxZ = parsed;
    }
    cur = cur.parentElement;
  }
  return maxZ;
}

function getTopmostModalDialog(): HTMLElement | null {
  const modalDialogs = Array.from(
    document.querySelectorAll<HTMLElement>('[role="dialog"][aria-modal="true"]'),
  );
  if (modalDialogs.length === 0) return null;
  let topmost = modalDialogs[0];
  let highestZ = getEffectiveZIndex(topmost);
  for (let i = 1; i < modalDialogs.length; i++) {
    const candidate = modalDialogs[i];
    const z = getEffectiveZIndex(candidate);
    if (z >= highestZ) {
      highestZ = z;
      topmost = candidate;
    }
  }
  return topmost;
}

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

      // 複数ダイアログが重なった場合は z-index（同値時は DOM 後順）で最前面のモーダルのみがキー操作を処理する
      const topmost = getTopmostModalDialog();
      if (topmost !== null && topmost !== currentDialog) {
        return;
      }

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

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      if (previousActive?.isConnected) {
        previousActive.focus();
      }
    };
  }, [dialogRef]);
}
