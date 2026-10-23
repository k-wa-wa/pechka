import { act, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { LanguageProvider, useLanguage } from "./LanguageContext";

function TestComponent() {
  const { language, setLanguage, t } = useLanguage();
  return (
    <div>
      <span data-testid="lang">{language}</span>
      <span data-testid="title">{t("admin.title")}</span>
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

    act(() => {
      screen.getByText("Switch JA").click();
    });

    expect(screen.getByTestId("lang").textContent).toBe("ja");
    expect(screen.getByTestId("title").textContent).toBe("管理画面");
  });
});
