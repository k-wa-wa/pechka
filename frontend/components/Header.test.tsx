import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import { ADMIN_CONTENTS } from "@/mocks/fixtures";
import EditModal from "./EditModal";
import Header from "./Header";

function EditWithHeaderHarness() {
  const [editing, setEditing] = useState(true);
  return (
    <LanguageProvider>
      <Header />
      {editing && (
        <EditModal content={ADMIN_CONTENTS[0]} onClose={() => setEditing(false)} onSave={vi.fn()} />
      )}
    </LanguageProvider>
  );
}

function SearchFirstHarness({ onEditClose }: { onEditClose: () => void }) {
  const [showEdit, setShowEdit] = useState(false);
  return (
    <LanguageProvider>
      <Header />
      <button type="button" onClick={() => setShowEdit(true)}>
        Underlying Edit
      </button>
      {showEdit && <EditModal content={ADMIN_CONTENTS[0]} onClose={onEditClose} onSave={vi.fn()} />}
    </LanguageProvider>
  );
}

describe("Header", () => {
  it("renders header elements and logo", () => {
    render(
      <LanguageProvider>
        <Header />
      </LanguageProvider>,
    );

    expect(screen.getByText("pechka")).toBeDefined();
  });

  it("opens and closes SearchModal with Ctrl/Cmd+K when no modal dialog is open", () => {
    render(
      <LanguageProvider>
        <Header />
      </LanguageProvider>,
    );

    fireEvent.keyDown(window, { key: "k", ctrlKey: true });
    expect(screen.getByRole("dialog", { name: "Search" })).toBeDefined();

    fireEvent.keyDown(window, { key: "Escape" });
    expect(screen.queryByRole("dialog", { name: "Search" })).toBeNull();
  });

  it("suppresses Ctrl/Cmd+K search shortcut while EditModal is open and preserves unsaved input", () => {
    render(<EditWithHeaderHarness />);
    const titleInput = screen.getByDisplayValue(ADMIN_CONTENTS[0].title) as HTMLInputElement;
    fireEvent.change(titleInput, { target: { value: "Unsaved Draft Title" } });

    fireEvent.keyDown(window, { key: "k", ctrlKey: true });
    fireEvent.keyDown(window, { key: "k", metaKey: true });
    expect(screen.queryByRole("dialog", { name: "Search" })).toBeNull();

    fireEvent.keyDown(document, { key: "Tab" });
    const editDialog = screen.getByRole("dialog", { name: "Edit Content" });
    expect(editDialog.contains(document.activeElement)).toBe(true);
    expect((screen.getByDisplayValue("Unsaved Draft Title") as HTMLInputElement).value).toBe(
      "Unsaved Draft Title",
    );
  });

  it("keeps Tab trapped and Escape scoped to SearchModal when SearchModal opens before underlying EditModal", () => {
    const onEditClose = vi.fn();
    render(<SearchFirstHarness onEditClose={onEditClose} />);
    // 1. 検索ダイアログを先に開く
    fireEvent.click(screen.getByTitle(/Search/));
    const searchDialog = screen.getByRole("dialog", { name: "Search" });
    const underlyingEditBtn = screen.getByText("Underlying Edit");
    expect(searchDialog.contains(document.activeElement)).toBe(true);

    // 2. Tab / Shift+Tab でも背後の管理ボタンにフォーカスが移らない
    fireEvent.keyDown(document, { key: "Tab" });
    expect(searchDialog.contains(document.activeElement)).toBe(true);
    expect(document.activeElement).not.toBe(underlyingEditBtn);

    // 3. 背後に編集ダイアログが同時に存在しても、z-index が高い検索側のみが Escape / Tab を処理する
    fireEvent.click(underlyingEditBtn);
    expect(screen.getByRole("dialog", { name: "Edit Content" })).toBeDefined();

    fireEvent.keyDown(document, { key: "Tab" });
    expect(searchDialog.contains(document.activeElement)).toBe(true);

    fireEvent.keyDown(document, { key: "Escape" });
    expect(screen.queryByRole("dialog", { name: "Search" })).toBeNull();
    expect(onEditClose).not.toHaveBeenCalled();
    expect(screen.getByRole("dialog", { name: "Edit Content" })).toBeDefined();
  });
});
