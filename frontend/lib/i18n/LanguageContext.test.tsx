import { act, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { LanguageProvider, useLanguage } from "./LanguageContext";

function TestComponent() {
  const { language, setLanguage, t } = useLanguage();
  return (
    <div>
      <span data-testid="lang">{language}</span>
      <span data-testid="title">{t("admin.title")}</span>
      <span data-testid="sort-label">{t("admin.sort.label")}</span>
      <span data-testid="sort-updated-desc">{t("admin.sort.updatedDesc")}</span>
      <span data-testid="sort-title-asc">{t("admin.sort.titleAsc")}</span>
      <button onClick={() => setLanguage("en")}>Switch EN</button>
      <button onClick={() => setLanguage("ja")}>Switch JA</button>
    </div>
  );
}

describe("LanguageContext", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("provides translations and allows switching language", () => {
    render(
      <LanguageProvider>
        <TestComponent />
      </LanguageProvider>,
    );

    expect(screen.getByTestId("lang").textContent).toBe("en");
    expect(screen.getByTestId("title").textContent).toBe("Admin Panel");
    expect(screen.getByTestId("sort-label").textContent).toBe("Sort by");
    expect(screen.getByTestId("sort-updated-desc").textContent).toBe("Updated (Newest)");
    expect(screen.getByTestId("sort-title-asc").textContent).toBe("Title (A–Z)");

    act(() => {
      screen.getByText("Switch JA").click();
    });

    expect(screen.getByTestId("lang").textContent).toBe("ja");
    expect(screen.getByTestId("title").textContent).toBe("管理画面");
    expect(screen.getByTestId("sort-label").textContent).toBe("並び替え");
    expect(screen.getByTestId("sort-updated-desc").textContent).toBe("更新日時（新しい順）");
    expect(screen.getByTestId("sort-title-asc").textContent).toBe("タイトル（昇順）");
  });
});
