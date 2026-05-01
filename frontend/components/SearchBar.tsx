"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";

export function SearchBar() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [q, setQ] = useState(searchParams.get("q") ?? "");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (q.trim()) router.push(`/search?q=${encodeURIComponent(q.trim())}`);
  }

  return (
    <form onSubmit={handleSubmit} className="flex gap-2">
      <input
        type="search"
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder="タイトル・タグで検索..."
        className="flex-1 rounded-lg px-3 py-2 text-sm focus:outline-none"
        style={{
          background: "var(--gh-surface)",
          border: "1px solid var(--gh-border)",
          color: "var(--gh-text)",
        }}
      />
      <button
        type="submit"
        className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity hover:opacity-80"
        style={{ background: "var(--nf-red)", color: "#fff" }}
      >
        検索
      </button>
    </form>
  );
}
