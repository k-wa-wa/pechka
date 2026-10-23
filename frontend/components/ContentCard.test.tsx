import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import { CONTENTS } from "@/mocks/fixtures";
import ContentCard from "./ContentCard";

describe("ContentCard", () => {
  it("renders content card information", () => {
    render(
      <LanguageProvider>
        <ContentCard content={CONTENTS[0]} />
      </LanguageProvider>,
    );

    expect(screen.getByText(CONTENTS[0].title)).toBeDefined();
  });
});
