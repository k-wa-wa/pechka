"use client";

import Link from "next/link";
import { useState } from "react";
import { type Content, type Disc, api } from "@/lib/api";
import { formatDuration } from "@/lib/utils";

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
    return <p className="text-gray-500 text-center py-12">コンテンツがありません</p>;
  }

  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="border-b border-gray-200 bg-gray-50">
          <th className="text-left px-4 py-3 font-medium text-gray-700">タイトル</th>
          <th className="text-left px-4 py-3 font-medium text-gray-700">種別</th>
          <th className="text-left px-4 py-3 font-medium text-gray-700">ディスク</th>
          <th className="text-left px-4 py-3 font-medium text-gray-700">尺</th>
          <th className="text-left px-4 py-3 font-medium text-gray-700">ステータス</th>
          <th className="text-left px-4 py-3 font-medium text-gray-700">操作</th>
        </tr>
      </thead>
      <tbody>
        {contents.map((c) => (
          <tr key={c.id} className="border-b border-gray-100 hover:bg-gray-50">
            <td className="px-4 py-3">
              <Link href={`/contents/${c.short_id}`} className="text-blue-600 hover:underline">
                {c.title}
              </Link>
            </td>
            <td className="px-4 py-3 text-gray-600">{c.content_type}</td>
            <td className="px-4 py-3 text-gray-600">{c.disc_id ? discMap[c.disc_id] ?? "-" : "-"}</td>
            <td className="px-4 py-3 text-gray-600">{formatDuration(c.duration_seconds)}</td>
            <td className="px-4 py-3">
              <span className={`px-2 py-0.5 rounded text-xs ${
                c.status === "ready" ? "bg-green-100 text-green-700" :
                c.status === "error" ? "bg-red-100 text-red-700" :
                "bg-yellow-100 text-yellow-700"
              }`}>
                {c.status}
              </span>
            </td>
            <td className="px-4 py-3">
              <div className="flex gap-2">
                <Link
                  href={`/admin/contents/${c.id}`}
                  className="text-blue-600 hover:underline"
                >
                  編集
                </Link>
                <button
                  onClick={() => handleDelete(c.id, c.title)}
                  className="text-red-600 hover:underline"
                >
                  削除
                </button>
              </div>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
