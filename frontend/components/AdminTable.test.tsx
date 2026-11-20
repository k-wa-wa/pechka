import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import * as api from "@/lib/api";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import { ADMIN_CONTENTS, CHANNELS } from "@/mocks/fixtures";
import AdminTable from "./AdminTable";

describe("AdminTable", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it("renders admin table headers and contents", () => {
    render(
      <LanguageProvider>
        <AdminTable initialContents={ADMIN_CONTENTS} channels={CHANNELS} />
      </LanguageProvider>,
    );

    expect(screen.getByText(ADMIN_CONTENTS[0].title)).toBeDefined();
    expect(screen.queryByLabelText("Sort by")).toBeNull();
  });

  it("renders sort select when onSortChange is provided and invokes callback", () => {
    const onSortChange = vi.fn();
    render(
      <LanguageProvider>
        <AdminTable
          initialContents={ADMIN_CONTENTS}
          channels={CHANNELS}
          sort="updated_desc"
          onSortChange={onSortChange}
        />
      </LanguageProvider>,
    );

    const select = screen.getByLabelText("Sort by") as HTMLSelectElement;
    expect(select.value).toBe("updated_desc");
    expect(select.options).toHaveLength(6);

    fireEvent.change(select, { target: { value: "title_asc" } });
    expect(onSortChange).toHaveBeenCalledWith("title_asc");
  });

  it("disables sort select when sortDisabled is true", () => {
    render(
      <LanguageProvider>
        <AdminTable
          initialContents={ADMIN_CONTENTS}
          channels={CHANNELS}
          sort="updated_desc"
          onSortChange={vi.fn()}
          sortDisabled={true}
        />
      </LanguageProvider>,
    );

    const select = screen.getByLabelText("Sort by") as HTMLSelectElement;
    expect(select.disabled).toBe(true);
  });

  it("polls in-progress rows without offset and preserves row order", async () => {
    vi.useFakeTimers();
    const pendingItem = {
      ...ADMIN_CONTENTS[0],
      id: "p1",
      status: "processing" as const,
      title: "Pending",
    };
    const readyItem = { ...ADMIN_CONTENTS[1], id: "r1", status: "ready" as const, title: "Ready" };

    const getSpy = vi.spyOn(api, "getAdminContents").mockResolvedValue({
      contents: [
        { ...readyItem, title: "Ready" },
        { ...pendingItem, status: "ready", title: "Pending Done" },
      ],
      total: 2,
      limit: 100,
      next_cursor: null,
      prev_cursor: null,
      has_next: false,
      has_prev: false,
    });

    render(
      <LanguageProvider>
        <AdminTable initialContents={[pendingItem, readyItem]} channels={CHANNELS} />
      </LanguageProvider>,
    );

    await vi.advanceTimersByTimeAsync(5000);

    expect(getSpy).toHaveBeenCalledWith({ limit: 100 });
    vi.useRealTimers();

    await waitFor(() => {
      const cells = screen.getAllByText(/Pending Done|Ready/);
      expect(cells[0].textContent).toBe("Pending Done");
    });
  });
});
