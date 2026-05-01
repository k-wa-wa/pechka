"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { type Content, type Disc, type ContentStatus, api } from "@/lib/api";

interface Props {
  content: Content;
  discs: Disc[];
}

export function ContentForm({ content, discs: _discs }: Props) {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const [title, setTitle] = useState(content.title);
  const [description, setDescription] = useState(content.description);
  const [tags, setTags] = useState(content.tags.join(", "));
  const [status, setStatus] = useState<ContentStatus>(content.status);

  const inputStyle = {
    background: "var(--gh-surface)",
    border: "1px solid var(--gh-border)",
    color: "var(--gh-text)",
    borderRadius: "6px",
    padding: "8px 12px",
    fontSize: "14px",
    width: "100%",
    outline: "none",
  };

  const labelStyle = { color: "var(--gh-muted)", fontSize: "14px", fontWeight: "500", marginBottom: "4px", display: "block" };

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const parsedTags = tags.split(",").map((t) => t.trim()).filter(Boolean);
      await api.admin.contents.update(content.id, {
        title: title || null,
        description: description || null,
        tags: parsedTags,
        status,
      });
      router.push("/admin");
      router.refresh();
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="rounded-lg p-6 space-y-4" style={{ background: "var(--gh-surface)", border: "1px solid var(--gh-border)" }}>
      {error && <p className="text-sm" style={{ color: "#f85149" }}>{error}</p>}

      <div>
        <label style={labelStyle}>タイトル *</label>
        <input
          type="text"
          required
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          style={inputStyle}
        />
      </div>

      <div>
        <label style={labelStyle}>説明</label>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={3}
          style={{ ...inputStyle, resize: "vertical" }}
        />
      </div>

      <div>
        <label style={labelStyle}>タグ（カンマ区切り）</label>
        <input
          type="text"
          value={tags}
          onChange={(e) => setTags(e.target.value)}
          placeholder="action, 4k, movie"
          style={inputStyle}
        />
      </div>

      <div>
        <label style={labelStyle}>ステータス</label>
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value as ContentStatus)}
          style={inputStyle}
        >
          <option value="pending">pending</option>
          <option value="processing">processing</option>
          <option value="ready">ready</option>
          <option value="error">error</option>
        </select>
      </div>

      <div className="flex gap-3 pt-2">
        <button
          type="submit"
          disabled={loading}
          className="px-5 py-2 rounded-lg text-sm font-medium transition-opacity hover:opacity-80 disabled:opacity-40"
          style={{ background: "var(--nf-red)", color: "#fff" }}
        >
          {loading ? "保存中..." : "保存"}
        </button>
        <button
          type="button"
          onClick={() => router.back()}
          className="px-5 py-2 rounded-lg text-sm font-medium transition-colors hover:opacity-80"
          style={{ background: "transparent", border: "1px solid var(--gh-border)", color: "var(--gh-text)" }}
        >
          キャンセル
        </button>
      </div>
    </form>
  );
}
