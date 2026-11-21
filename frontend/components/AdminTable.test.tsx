import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
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

  it("does not revert recently edited row when poll returns pre-sync response and stops polling", async () => {
    vi.useFakeTimers();
    const pendingItem = {
      ...ADMIN_CONTENTS[0],
      id: "p1",
      status: "processing" as const,
      title: "Processing Video",
      updated_at: "2025-01-01T00:00:00Z",
    };
    const readyItem = {
      ...ADMIN_CONTENTS[1],
      id: "r1",
      status: "ready" as const,
      title: "Original Title",
      updated_at: "2025-01-01T00:00:00Z",
    };

    const originalFetch = global.fetch;
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        ...readyItem,
        title: "Saved Title",
        updated_at: "2025-01-01T00:01:00Z",
      }),
    }) as typeof fetch;

    const getSpy = vi.spyOn(api, "getAdminContents").mockResolvedValue({
      contents: [
        {
          ...pendingItem,
          status: "ready",
          title: "Processing Video Done",
          updated_at: "2025-01-01T00:01:05Z",
        },
        // Stale pre-sync MongoDB document for r1
        { ...readyItem, title: "Original Title", updated_at: "2025-01-01T00:00:00Z" },
      ],
      total: 2,
      limit: 100,
      next_cursor: null,
      prev_cursor: null,
      has_next: false,
      has_prev: false,
    });

    try {
      render(
        <LanguageProvider>
          <AdminTable initialContents={[pendingItem, readyItem]} channels={CHANNELS} />
        </LanguageProvider>,
      );

      // Edit r1 (second row)
      const editButtons = screen.getAllByRole("button", { name: "Edit" });
      fireEvent.click(editButtons[1]);
      const titleInput = screen.getByDisplayValue("Original Title");
      fireEvent.change(titleInput, { target: { value: "Saved Title" } });
      await act(async () => {
        fireEvent.click(screen.getByRole("button", { name: "Save" }));
        await Promise.resolve();
        await Promise.resolve();
      });

      expect(screen.getByText("Saved Title")).toBeDefined();

      // Poll fires and flips the last processing row to ready while returning stale r1
      await act(async () => {
        await vi.advanceTimersByTimeAsync(5000);
        await Promise.resolve();
      });
      expect(getSpy).toHaveBeenCalledTimes(1);

      expect(screen.getByText("Processing Video Done")).toBeDefined();
      expect(screen.getByText("Saved Title")).toBeDefined();
      expect(screen.queryByText("Original Title")).toBeNull();

      // Polling stops because no row remains pending/processing
      await act(async () => {
        await vi.advanceTimersByTimeAsync(5000);
      });
      expect(getSpy).toHaveBeenCalledTimes(1);
    } finally {
      global.fetch = originalFetch;
    }
  });

  it("does not revert archived state on stale poll, but adopts newer post-sync poll", async () => {
    vi.useFakeTimers();
    const pendingItem = {
      ...ADMIN_CONTENTS[0],
      id: "p1",
      status: "processing" as const,
      title: "Processing Video",
      updated_at: "2025-01-01T00:00:00Z",
    };
    const readyItem = {
      ...ADMIN_CONTENTS[1],
      id: "r1",
      status: "ready" as const,
      title: "Target Video",
      archived_at: null,
      updated_at: "2025-01-01T00:00:00Z",
    };

    vi.spyOn(window, "confirm").mockReturnValue(true);
    vi.spyOn(api, "archiveContent").mockResolvedValue({
      ...readyItem,
      archived_at: "2025-01-01T00:01:00Z",
      updated_at: "2025-01-01T00:01:00Z",
    });

    const getSpy = vi
      .spyOn(api, "getAdminContents")
      // First poll: stale pre-sync MongoDB response while p1 stays processing
      .mockResolvedValueOnce({
        contents: [
          { ...pendingItem },
          { ...readyItem, archived_at: null, updated_at: "2025-01-01T00:00:00Z" },
        ],
        total: 2,
        limit: 100,
        next_cursor: null,
        prev_cursor: null,
        has_next: false,
        has_prev: false,
      })
      // Second poll: newer update after sync
      .mockResolvedValueOnce({
        contents: [
          { ...pendingItem, status: "ready", updated_at: "2025-01-01T00:02:00Z" },
          {
            ...readyItem,
            title: "Target Video Synced",
            archived_at: "2025-01-01T00:01:00Z",
            updated_at: "2025-01-01T00:02:00Z",
          },
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

    const archiveButtons = screen.getAllByRole("button", { name: "Archive" });
    await act(async () => {
      fireEvent.click(archiveButtons[1]);
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(screen.getByText("Archived")).toBeDefined();

    // First poll: stale response must not revert Archived badge
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
      await Promise.resolve();
    });
    expect(getSpy).toHaveBeenCalledTimes(1);
    expect(screen.getByText("Archived")).toBeDefined();

    // Second poll: newer updated_at is adopted
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
      await Promise.resolve();
    });
    expect(getSpy).toHaveBeenCalledTimes(2);
    expect(screen.getByText("Target Video Synced")).toBeDefined();
    expect(screen.getByText("Archived")).toBeDefined();
  });
});
