import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import { ADMIN_CONTENTS, CHANNELS } from "@/mocks/fixtures";
import AdminTable from "./AdminTable";

describe("AdminTable", () => {
  it("renders admin table headers and contents", () => {
    render(
      <LanguageProvider>
        <AdminTable initialContents={ADMIN_CONTENTS} channels={CHANNELS} />
      </LanguageProvider>,
    );

    expect(screen.getByText(ADMIN_CONTENTS[0].title)).toBeDefined();
  });
});
