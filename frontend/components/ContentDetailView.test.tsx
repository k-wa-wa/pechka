import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import { CONTENTS } from "@/mocks/fixtures";
import ContentDetailView from "./ContentDetailView";

describe("ContentDetailView", () => {
  it("renders content detail title", () => {
    render(
      <LanguageProvider>
        <ContentDetailView content={CONTENTS[0]} variants={[]} />
      </LanguageProvider>,
    );

    expect(screen.getByText(CONTENTS[0].title)).toBeDefined();
  });
});
