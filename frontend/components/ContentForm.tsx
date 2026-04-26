"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { type Content, type Disc, type ContentType, type ContentStatus, api } from "@/lib/api";

interface Props {
  content?: Content;
  discs: Disc[];
}

export function ContentForm({ content, discs }: Props) {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const [title, setTitle] = useState(content?.title ?? "");
  const [description, setDescription] = useState(content?.description ?? "");
  const [contentType, setContentType] = useState<ContentType>(content?.content_type ?? "video");
  const [discId, setDiscId] = useState(content?.disc_id ?? "");
  const [durationSeconds, setDurationSeconds] = useState<string>(content?.duration_seconds?.toString() ?? "");
  const [is360, setIs360] = useState(content?.is_360 ?? false);
  const [tags, setTags] = useState(content?.tags.join(", ") ?? "");
  const [status, setStatus] = useState<ContentStatus>(content?.status ?? "pending");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const parsedTags = tags.split(",").map((t) => t.trim()).filter(Boolean);
      if (content) {
        await api.admin.contents.update(content.id, {
          title: title || null,
          description: description || null,
          tags: parsedTags,
          status,
        });
      } else {
        await api.admin.contents.create({
          content_type: contentType,
          disc_id: discId || null,
          title,
          description,
          duration_seconds: durationSeconds ? parseInt(durationSeconds) : null,
          is_360: is360,
          tags: parsedTags,
        });
      }
      router.push("/admin");
      router.refresh();
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="bg-white border border-gray-200 rounded-lg p-6 space-y-4">
      {error && <p className="text-red-600 text-sm">{error}</p>}

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">タイトル *</label>
        <input
          type="text"
          required
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">説明</label>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={3}
          className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      {!content && (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">種別 *</label>
            <select
              value={contentType}
              onChange={(e) => setContentType(e.target.value as ContentType)}
              className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="video">動画</option>
              <option value="image_gallery">画像ギャラリー</option>
              <option value="vr360">360° VR</option>
              <option value="document">ドキュメント</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">ディスク</label>
            <select
              value={discId}
              onChange={(e) => setDiscId(e.target.value)}
              className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">なし</option>
              {discs.map((d) => (
                <option key={d.id} value={d.id}>{d.label}{d.disc_name ? ` (${d.disc_name})` : ""}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">尺（秒）</label>
            <input
              type="number"
              value={durationSeconds}
              onChange={(e) => setDurationSeconds(e.target.value)}
              className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="is360"
              checked={is360}
              onChange={(e) => setIs360(e.target.checked)}
              className="rounded"
            />
            <label htmlFor="is360" className="text-sm text-gray-700">360° VR動画</label>
          </div>
        </>
      )}

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">タグ（カンマ区切り）</label>
        <input
          type="text"
          value={tags}
          onChange={(e) => setTags(e.target.value)}
          placeholder="action, 4k, movie"
          className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      {content && (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">ステータス</label>
          <select
            value={status}
            onChange={(e) => setStatus(e.target.value as ContentStatus)}
            className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="pending">pending</option>
            <option value="processing">processing</option>
            <option value="ready">ready</option>
            <option value="error">error</option>
          </select>
        </div>
      )}

      <div className="flex gap-3 pt-2">
        <button
          type="submit"
          disabled={loading}
          className="bg-blue-600 text-white px-5 py-2 rounded-lg text-sm hover:bg-blue-700 transition-colors disabled:opacity-50"
        >
          {loading ? "保存中..." : "保存"}
        </button>
        <button
          type="button"
          onClick={() => router.back()}
          className="border border-gray-300 text-gray-700 px-5 py-2 rounded-lg text-sm hover:bg-gray-50 transition-colors"
        >
          キャンセル
        </button>
      </div>
    </form>
  );
}
