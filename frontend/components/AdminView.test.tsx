import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import { ADMIN_CONTENTS, CHANNELS } from "@/mocks/fixtures";
import AdminView from "./AdminView";

describe("AdminView", () => {
  it("renders admin view with initial contents", () => {
    render(
      <LanguageProvider>
        <AdminView
          contents={ADMIN_CONTENTS}
          total={ADMIN_CONTENTS.length}
          limit={10}
          offset={0}
          channels={CHANNELS}
        />
      </LanguageProvider>,
    );

    expect(screen.getByText(ADMIN_CONTENTS[0].title)).toBeDefined();
  });
});
