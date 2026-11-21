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
});
