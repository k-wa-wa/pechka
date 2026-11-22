"use client";

import { useState } from "react";
import AdminTable from "@/components/AdminTable";
import { getAdminContents } from "@/lib/api";
import { useLanguage } from "@/lib/i18n/LanguageContext";
import type { AdminContentSort, Channel, Content } from "@/lib/types";

interface Props {
  contents: Content[];
  total: number;
  limit: number;
  nextCursor: string | null;
  prevCursor: string | null;
  hasNext: boolean;
  hasPrev: boolean;
  channels?: Channel[];
}

export default function AdminView({
  contents,
  total,
  limit,
  nextCursor,
  prevCursor,
  hasNext,
  hasPrev,
  channels,
}: Props) {
  const { t } = useLanguage();
  const [items, setItems] = useState(contents);
  const [currentTotal, setCurrentTotal] = useState(total);
  const [currentSort, setCurrentSort] = useState<AdminContentSort>("updated_desc");
  const [currentNextCursor, setCurrentNextCursor] = useState<string | null>(nextCursor);
  const [currentPrevCursor, setCurrentPrevCursor] = useState<string | null>(prevCursor);
  const [currentHasNext, setCurrentHasNext] = useState(hasNext);
  const [currentHasPrev, setCurrentHasPrev] = useState(hasPrev);
  const [currentPage, setCurrentPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);

  const totalPages = Math.max(1, Math.ceil(currentTotal / limit));

  async function applyPage(
    res: Awaited<ReturnType<typeof getAdminContents>>,
    nextSort: AdminContentSort,
    nextPage: number,
  ) {
    setItems(res.contents);
    setCurrentTotal(res.total);
    setCurrentSort(nextSort);
    setCurrentNextCursor(res.next_cursor);
    setCurrentPrevCursor(res.prev_cursor);
    setCurrentHasNext(res.has_next);
    setCurrentHasPrev(res.has_prev);
    setCurrentPage(nextPage);
  }

  async function handleNext() {
    if (loading || !currentHasNext || !currentNextCursor) return;

    setLoading(true);
    setError(false);
    try {
      const res = await getAdminContents({ limit, sort: currentSort, after: currentNextCursor });
      if (res.contents.length === 0) {
        const first = await getAdminContents({ limit, sort: currentSort });
        await applyPage(first, currentSort, 1);
      } else {
        await applyPage(res, currentSort, currentPage + 1);
      }
    } catch {
      setError(true);
    } finally {
      setLoading(false);
    }
  }

  async function handlePrev() {
    if (loading || !currentHasPrev || !currentPrevCursor) return;

    setLoading(true);
    setError(false);
    try {
      const res = await getAdminContents({ limit, sort: currentSort, before: currentPrevCursor });
      if (res.contents.length === 0) {
        const first = await getAdminContents({ limit, sort: currentSort });
        await applyPage(first, currentSort, 1);
      } else {
        await applyPage(res, currentSort, res.has_prev ? Math.max(1, currentPage - 1) : 1);
      }
    } catch {
      setError(true);
    } finally {
      setLoading(false);
    }
  }

  async function handleSortChange(nextSort: AdminContentSort) {
    if (loading) return;

    setLoading(true);
    setError(false);
    try {
      const res = await getAdminContents({ limit, sort: nextSort });
      await applyPage(res, nextSort, 1);
    } catch {
      setError(true);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div
      style={{
        maxWidth: 1280,
        margin: "0 auto",
        padding: "32px 24px",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: 24,
        }}
      >
        <h1
          style={{
            margin: 0,
            fontSize: 22,
            fontWeight: 700,
            color: "var(--fg)",
          }}
        >
          {t("admin.title")}
        </h1>
        <span style={{ fontSize: 14, color: "var(--muted)" }}>
          {currentTotal} {t("admin.itemsCount")}
        </span>
      </div>

      <AdminTable
        initialContents={items}
        channels={channels}
        sort={currentSort}
        onSortChange={handleSortChange}
        sortDisabled={loading}
        onUploaded={() => setCurrentTotal((t) => t + 1)}
      />

      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          gap: 12,
          marginTop: 20,
        }}
      >
        <button
          onClick={handlePrev}
          disabled={loading || !currentHasPrev}
          style={{
            padding: "6px 14px",
            borderRadius: "var(--radius-md)",
            border: "1px solid var(--line)",
            backgroundColor: "transparent",
            color: !currentHasPrev ? "var(--muted-border)" : "var(--fg)",
            cursor: loading || !currentHasPrev ? "not-allowed" : "pointer",
            fontSize: 13,
            transition: "all 0.15s",
          }}
        >
          {t("admin.pagination.prev")}
        </button>

        <span style={{ fontSize: 13, color: "var(--muted)" }}>
          {t("admin.pagination.pageLabel")} {currentPage} / {totalPages}
        </span>

        <button
          onClick={handleNext}
          disabled={loading || !currentHasNext}
          style={{
            padding: "6px 14px",
            borderRadius: "var(--radius-md)",
            border: "1px solid var(--line)",
            backgroundColor: "transparent",
            color: !currentHasNext ? "var(--muted-border)" : "var(--fg)",
            cursor: loading || !currentHasNext ? "not-allowed" : "pointer",
            fontSize: 13,
            transition: "all 0.15s",
          }}
        >
          {t("admin.pagination.next")}
        </button>
      </div>

      {error && (
        <div
          style={{
            textAlign: "center",
            marginTop: 12,
            fontSize: 13,
            color: "var(--warn)",
          }}
        >
          {t("admin.pagination.loadError")}
        </div>
      )}
    </div>
  );
}
