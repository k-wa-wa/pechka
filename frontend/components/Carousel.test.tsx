import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import { CONTENTS } from "@/mocks/fixtures";
import Carousel from "./Carousel";

describe("Carousel", () => {
  it("renders carousel items", () => {
    render(
      <LanguageProvider>
        <Carousel items={CONTENTS} />
      </LanguageProvider>,
    );

    expect(screen.getByText(CONTENTS[0].title)).toBeDefined();
  });
});
