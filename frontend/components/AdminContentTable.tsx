"use client";

import Link from "next/link";
import { useState } from "react";
import { type Content, type Disc, api } from "@/lib/api";
import { formatDuration } from "@/lib/utils";

const STATUS_STYLE: Record<string, { bg: string; text: string }> = {
  ready: { bg: "#1a3a2a", text: "#3fb950" },
  processing: { bg: "#332900", text: "#d29922" },
  error: { bg: "#3d0d0d", text: "#f85149" },
  pending: { bg: "#21262d", text: "#8b949e" },
};

const TYPE_LABELS: Record<string, string> = {
  video: "動画",
  vr360: "VR",
  image_gallery: "画像",
  document: "ドキュメント",
};

interface Props {
  contents: Content[];
  discs: Disc[];
}

export function AdminContentTable({ contents: initial, discs }: Props) {
  const [contents, setContents] = useState(initial);
  const discMap = Object.fromEntries(discs.map((d) => [d.id, d.label]));

  async function handleDelete(id: string, title: string) {
    if (!confirm(`「${title}」を削除しますか？`)) return;
    await api.admin.contents.delete(id);
    setContents((prev) => prev.filter((c) => c.id !== id));
  }

  if (contents.length === 0) {
    return <p className="text-center py-12" style={{ color: "var(--gh-muted)" }}>コンテンツがありません（ETL パイプライン経由で追加されます）</p>;
  }

  return (
    <table className="w-full text-sm">
      <thead>
        <tr style={{ borderBottom: "1px solid var(--gh-border)", background: "#161b22" }}>
          <th className="text-left px-4 py-3 font-medium" style={{ color: "var(--gh-muted)" }}>タイトル</th>
          <th className="text-left px-4 py-3 font-medium" style={{ color: "var(--gh-muted)" }}>種別</th>
          <th className="text-left px-4 py-3 font-medium" style={{ color: "var(--gh-muted)" }}>ディスク</th>
          <th className="text-left px-4 py-3 font-medium" style={{ color: "var(--gh-muted)" }}>尺</th>
          <th className="text-left px-4 py-3 font-medium" style={{ color: "var(--gh-muted)" }}>ステータス</th>
          <th className="text-left px-4 py-3 font-medium" style={{ color: "var(--gh-muted)" }}>操作</th>
        </tr>
      </thead>
      <tbody>
        {contents.map((c) => {
          const s = STATUS_STYLE[c.status] ?? STATUS_STYLE.pending;
          return (
            <tr key={c.id} className="transition-colors hover:opacity-80" style={{ borderBottom: "1px solid var(--gh-border)" }}>
              <td className="px-4 py-3">
                <Link href={`/contents/${c.short_id}`} style={{ color: "var(--gh-accent)" }} className="hover:underline">
                  {c.title}
                </Link>
              </td>
              <td className="px-4 py-3" style={{ color: "var(--gh-muted)" }}>{TYPE_LABELS[c.content_type] ?? c.content_type}</td>
              <td className="px-4 py-3" style={{ color: "var(--gh-muted)" }}>{c.disc_id ? discMap[c.disc_id] ?? "-" : "-"}</td>
              <td className="px-4 py-3 font-mono" style={{ color: "var(--gh-muted)" }}>{formatDuration(c.duration_seconds)}</td>
              <td className="px-4 py-3">
                <span className="px-2 py-0.5 rounded text-xs font-medium" style={{ background: s.bg, color: s.text }}>
                  {c.status}
                </span>
              </td>
              <td className="px-4 py-3">
                <div className="flex gap-3">
                  <Link href={`/admin/contents/${c.id}`} style={{ color: "var(--gh-accent)" }} className="hover:underline">
                    編集
                  </Link>
                  <button onClick={() => handleDelete(c.id, c.title)} style={{ color: "#f85149" }} className="hover:underline">
                    削除
                  </button>
                </div>
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
