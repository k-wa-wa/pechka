import { act, fireEvent, render, screen } from "@testing-library/react";
import { usePathname } from "next/navigation";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import Header from "./Header";

vi.mock("next/navigation", () => ({
  usePathname: vi.fn(() => "/"),
}));

function renderHeader() {
  return render(
    <LanguageProvider>
      <Header />
    </LanguageProvider>,
  );
}

describe("Header", () => {
  beforeEach(() => {
    vi.mocked(usePathname).mockReturnValue("/");
  });

  it("renders header elements and logo", () => {
    renderHeader();

    expect(screen.getByText("pechka")).toBeDefined();
  });

  it("closes the drawer when the pathname changes", () => {
    const { rerender } = renderHeader();
    fireEvent.click(screen.getByLabelText("Menu"));
    expect(document.getElementById("mobile-drawer")).not.toBeNull();

    vi.mocked(usePathname).mockReturnValue("/admin");
    rerender(
      <LanguageProvider>
        <Header />
      </LanguageProvider>,
    );

    expect(document.getElementById("mobile-drawer")).toBeNull();
  });

  it("closes the drawer on Escape only while open", () => {
    renderHeader();
    const button = screen.getByLabelText("Menu");

    fireEvent.keyDown(window, { key: "Escape" });
    expect(document.getElementById("mobile-drawer")).toBeNull();

    fireEvent.click(button);
    expect(document.getElementById("mobile-drawer")).not.toBeNull();
    act(() => {
      fireEvent.keyDown(window, { key: "Escape" });
    });
    expect(document.getElementById("mobile-drawer")).toBeNull();
  });

  it("exposes aria-expanded and aria-controls on the hamburger", () => {
    renderHeader();
    const button = screen.getByLabelText("Menu");

    expect(button.getAttribute("aria-controls")).toBe("mobile-drawer");
    expect(button.getAttribute("aria-expanded")).toBe("false");

    fireEvent.click(button);
    expect(button.getAttribute("aria-expanded")).toBe("true");
    expect(document.getElementById("mobile-drawer")).not.toBeNull();
  });
});
