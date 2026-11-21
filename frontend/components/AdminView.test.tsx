import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import * as api from "@/lib/api";
import { LanguageProvider } from "@/lib/i18n/LanguageContext";
import { ADMIN_CONTENTS, CHANNELS } from "@/mocks/fixtures";
import AdminView from "./AdminView";

describe("AdminView", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders admin view with initial contents and sort selector", () => {
    render(
      <LanguageProvider>
        <AdminView
          contents={ADMIN_CONTENTS}
          total={ADMIN_CONTENTS.length}
          limit={10}
          nextCursor={null}
          prevCursor={null}
          hasNext={false}
          hasPrev={false}
          channels={CHANNELS}
        />
      </LanguageProvider>,
    );

    expect(screen.getByText(ADMIN_CONTENTS[0].title)).toBeDefined();
    expect((screen.getByLabelText("Sort by") as HTMLSelectElement).value).toBe("updated_desc");
  });

  it("navigates next/prev with cursors and resets to first page on sort change", async () => {
    const page1 = [ADMIN_CONTENTS[0]];
    const page2 = [ADMIN_CONTENTS[1]];
    const sortedPage1 = [ADMIN_CONTENTS[2]];

    const getSpy = vi
      .spyOn(api, "getAdminContents")
      .mockResolvedValueOnce({
        contents: page2,
        total: 3,
        limit: 1,
        next_cursor: "cur-2-next",
        prev_cursor: "cur-2-prev",
        has_next: true,
        has_prev: true,
      })
      .mockResolvedValueOnce({
        contents: page1,
        total: 3,
        limit: 1,
        next_cursor: "cur-1-next",
        prev_cursor: null,
        has_next: true,
        has_prev: false,
      })
      .mockResolvedValueOnce({
        contents: sortedPage1,
        total: 3,
        limit: 1,
        next_cursor: "cur-sorted-next",
        prev_cursor: null,
        has_next: true,
        has_prev: false,
      });

    render(
      <LanguageProvider>
        <AdminView
          contents={page1}
          total={3}
          limit={1}
          nextCursor="cur-1-next"
          prevCursor={null}
          hasNext={true}
          hasPrev={false}
          channels={CHANNELS}
        />
      </LanguageProvider>,
    );

    const prevBtn = screen.getByRole("button", { name: "Previous" }) as HTMLButtonElement;
    const nextBtn = screen.getByRole("button", { name: "Next" }) as HTMLButtonElement;
    expect(prevBtn.disabled).toBe(true);
    expect(nextBtn.disabled).toBe(false);

    // Next -> page 2
    fireEvent.click(nextBtn);
    await waitFor(() => expect(screen.getByText(page2[0].title)).toBeDefined());
    expect(getSpy).toHaveBeenNthCalledWith(1, {
      limit: 1,
      sort: "updated_desc",
      after: "cur-1-next",
    });
    expect(screen.getByText("Page 2 / 3")).toBeDefined();

    // Prev -> page 1
    fireEvent.click(screen.getByRole("button", { name: "Previous" }));
    await waitFor(() => expect(screen.getByText(page1[0].title)).toBeDefined());
    expect(getSpy).toHaveBeenNthCalledWith(2, {
      limit: 1,
      sort: "updated_desc",
      before: "cur-2-prev",
    });
    expect(screen.getByText("Page 1 / 3")).toBeDefined();

    // Sort change -> resets to first page without cursor
    fireEvent.change(screen.getByLabelText("Sort by"), { target: { value: "title_asc" } });
    await waitFor(() => expect(screen.getByText(sortedPage1[0].title)).toBeDefined());
    expect(getSpy).toHaveBeenNthCalledWith(3, {
      limit: 1,
      sort: "title_asc",
    });
    expect((screen.getByLabelText("Sort by") as HTMLSelectElement).value).toBe("title_asc");
    expect(screen.getByText("Page 1 / 3")).toBeDefined();
  });

  it("auto-recovers to first page when cursor transition returns empty contents", async () => {
    const page1 = [ADMIN_CONTENTS[0]];
    const getSpy = vi
      .spyOn(api, "getAdminContents")
      .mockResolvedValueOnce({
        contents: [],
        total: 1,
        limit: 1,
        next_cursor: null,
        prev_cursor: null,
        has_next: false,
        has_prev: false,
      })
      .mockResolvedValueOnce({
        contents: page1,
        total: 1,
        limit: 1,
        next_cursor: null,
        prev_cursor: null,
        has_next: false,
        has_prev: false,
      });

    render(
      <LanguageProvider>
        <AdminView
          contents={[ADMIN_CONTENTS[1]]}
          total={2}
          limit={1}
          nextCursor="stale-cursor"
          prevCursor={null}
          hasNext={true}
          hasPrev={false}
          channels={CHANNELS}
        />
      </LanguageProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    await waitFor(() => expect(screen.getByText(page1[0].title)).toBeDefined());

    expect(getSpy).toHaveBeenNthCalledWith(1, {
      limit: 1,
      sort: "updated_desc",
      after: "stale-cursor",
    });
    expect(getSpy).toHaveBeenNthCalledWith(2, {
      limit: 1,
      sort: "updated_desc",
    });
    expect(screen.getByText("Page 1 / 1")).toBeDefined();
  });

  it("preserves state, disables controls during load, and shows error on failure", async () => {
    let rejectFn: ((reason?: unknown) => void) | undefined;
    vi.spyOn(api, "getAdminContents").mockImplementation(
      () =>
        new Promise((_, reject) => {
          rejectFn = reject;
        }),
    );

    render(
      <LanguageProvider>
        <AdminView
          contents={[ADMIN_CONTENTS[0]]}
          total={2}
          limit={1}
          nextCursor="cur-1"
          prevCursor={null}
          hasNext={true}
          hasPrev={false}
          channels={CHANNELS}
        />
      </LanguageProvider>,
    );

    fireEvent.change(screen.getByLabelText("Sort by"), { target: { value: "title_desc" } });

    expect((screen.getByLabelText("Sort by") as HTMLSelectElement).disabled).toBe(true);
    expect((screen.getByRole("button", { name: "Next" }) as HTMLButtonElement).disabled).toBe(true);

    rejectFn?.(new Error("network error"));

    await waitFor(() => {
      expect(screen.getByText("Failed to load page")).toBeDefined();
    });
    // Sort and items remain unchanged on error
    expect((screen.getByLabelText("Sort by") as HTMLSelectElement).value).toBe("updated_desc");
    expect(screen.getByText(ADMIN_CONTENTS[0].title)).toBeDefined();
    expect((screen.getByLabelText("Sort by") as HTMLSelectElement).disabled).toBe(false);
  });
});
