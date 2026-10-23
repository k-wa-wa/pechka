import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import { ADMIN_CONTENTS } from "@/mocks/fixtures";
import EditModal from "./EditModal";

describe("EditModal", () => {
  it("renders edit modal form when open", () => {
    render(
      <LanguageProvider>
        <EditModal content={ADMIN_CONTENTS[0]} onClose={vi.fn()} onSave={vi.fn()} />
      </LanguageProvider>,
    );

    expect(screen.getByDisplayValue(ADMIN_CONTENTS[0].title)).toBeDefined();
  });
});
