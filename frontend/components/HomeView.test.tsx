import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import { CONTENTS } from "@/mocks/fixtures";
import HomeView from "./HomeView";

describe("HomeView", () => {
  it("renders contents in home view", () => {
    render(
      <LanguageProvider>
        <HomeView carouselReady={CONTENTS} gridItems={CONTENTS} currentType="" />
      </LanguageProvider>,
    );

    expect(screen.getByText("All")).toBeDefined();
  });
});
