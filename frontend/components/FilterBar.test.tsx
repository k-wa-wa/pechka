import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import FilterBar from "./FilterBar";

const mockTypes = [
  { value: "" as const, label: "All" },
  { value: "video" as const, label: "Video" },
];

describe("FilterBar", () => {
  it("renders filter buttons", () => {
    render(
      <LanguageProvider>
        <FilterBar types={mockTypes} currentType="" />
      </LanguageProvider>,
    );

    expect(screen.getByText("All")).toBeDefined();
  });
});
