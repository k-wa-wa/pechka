import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import Header from "./Header";

describe("Header", () => {
  it("renders header elements and logo", () => {
    render(
      <LanguageProvider>
        <Header />
      </LanguageProvider>,
    );

    expect(screen.getByText("pechka")).toBeDefined();
  });
});
