"use client";

import { useState, useEffect, use } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { apiClient } from "@/lib/api-client";

export default function EditContentPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const [content, setContent] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");

  const API_BASE = "/api/metadata/v1";

  useEffect(() => {
    async function fetchContent() {
      try {
        const res = await apiClient.get(`${API_BASE}/admin/metadata/contents/${id}`);
        setContent(res.data);
      } catch (error: any) {
        if (error.response?.status === 404) {
          setMessage("Content not found");
        } else {
          setMessage("Failed to load content data.");
        }
      } finally {
        setLoading(false);
      }
    }
    fetchContent();
  }, [id]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setMessage("");

    try {
      await apiClient.put(`${API_BASE}/admin/metadata/contents/${id}`, {
        title: content.title,
        description: content.description,
        rating: parseFloat(content.rating) || 0,
        content_type: content.content_type,
        tags: content.tags || [],
        visibility: content.visibility || "public",
        allowed_groups: content.allowed_groups || [],
      });
      setMessage("Content updated successfully! Redirecting...");
      setTimeout(() => router.push("/admin"), 1500);
    } catch (error: any) {
      setMessage(`Error: ${error.response?.data?.error || "Failed to update"}`);
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div className="min-h-screen bg-black flex items-center justify-center text-white font-black tracking-widest uppercase text-xs animate-pulse text-center">Loading Content...</div>;

  if (!content && !loading) return (
    <div className="min-h-screen bg-black flex flex-col items-center justify-center text-white gap-8 text-center p-4">
      <div className="space-y-2">
        <p className="text-white/20 text-sm font-black tracking-widest uppercase">Error Encountered</p>
        <p className="text-2xl font-black">{message || "Content not found"}</p>
      </div>
      <Link href="/admin" className="px-8 py-3 bg-white/10 hover:bg-white/20 rounded-full transition-all text-xs font-bold uppercase tracking-widest border border-white/10">
        Back to Dashboard
      </Link>
    </div>
  );

  return (
    <main className="min-h-screen bg-black pt-28 px-4 md:px-12 pb-24 text-white">
      <div className="max-w-4xl mx-auto">
        <header className="mb-16">
          <Link href="/admin" className="text-white/30 hover:text-white mb-6 inline-flex items-center gap-2 transition-all text-xs font-bold uppercase tracking-widest group">
            <span className="group-hover:-translate-x-1 transition-transform">←</span> Back to Dashboard
          </Link>
          <div className="flex flex-col md:flex-row md:items-end md:justify-between gap-6">
            <div>
              <h1 className="text-5xl md:text-6xl font-black tracking-tighter mb-4">Edit Metadata</h1>
              <div className="flex items-center gap-4">
                <span className="text-[10px] font-black uppercase tracking-widest bg-white/10 px-3 py-1 rounded-full text-white/60">ID: {content.short_id}</span>
                <span className="text-[10px] font-black uppercase tracking-widest bg-red-600/20 text-red-500 px-3 py-1 rounded-full border border-red-600/20">{content.content_type}</span>
              </div>
            </div>
          </div>
        </header>

        <form onSubmit={handleSubmit} className="space-y-12">
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-12">
            <div className="lg:col-span-2 space-y-10">
              <div className="space-y-4">
                <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Content Title</label>
                <input
                  type="text"
                  value={content.title}
                  onChange={(e) => setContent({...content, title: e.target.value})}
                  className="w-full bg-white/[0.03] border border-white/10 rounded-2xl px-6 py-4 text-xl font-bold text-white focus:outline-none focus:border-red-600 focus:bg-white/[0.05] transition-all placeholder:text-white/10"
                  placeholder="Enter content title..."
                  required
                />
              </div>

              <div className="space-y-4">
                <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Detailed Description</label>
                <textarea
                  value={content.description}
                  onChange={(e) => setContent({...content, description: e.target.value})}
                  rows={8}
                  className="w-full bg-white/[0.03] border border-white/10 rounded-2xl px-6 py-4 text-white font-medium focus:outline-none focus:border-red-600 focus:bg-white/[0.05] transition-all resize-none leading-relaxed placeholder:text-white/10"
                  placeholder="Tell the story of this content..."
                />
              </div>

              <div className="space-y-4">
                <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Tags (Comma separated)</label>
                <input
                  type="text"
                  value={(content.tags || []).join(", ")}
                  onChange={(e) => setContent({
                    ...content,
                    tags: e.target.value.split(",").map((t: string) => t.trim()).filter((t: string) => t !== "")
                  })}
                  className="w-full bg-white/[0.03] border border-white/10 rounded-2xl px-6 py-4 text-white font-medium focus:outline-none focus:border-red-600 focus:bg-white/[0.05] transition-all placeholder:text-white/10"
                  placeholder="Action, Sci-Fi, Adventure"
                />
              </div>
            </div>

            <div className="space-y-10">
              <div className="space-y-4">
                <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Preview Thumbnail</label>
                <div className="relative aspect-video rounded-2xl overflow-hidden bg-white/5 border border-white/10 group">
                  {content.assets?.find((a: any) => a.asset_role === "thumbnail")?.public_url ? (
                    <img
                      src={content.assets.find((a: any) => a.asset_role === "thumbnail").public_url}
                      alt={content.title}
                      className="object-cover w-full h-full group-hover:scale-105 transition-transform duration-700"
                    />
                  ) : (
                    <div className="w-full h-full flex flex-col items-center justify-center text-[10px] font-black uppercase tracking-widest text-white/20">
                      <span>No Thumbnail</span>
                      <span className="text-[32px] mt-2">∅</span>
                    </div>
                  )}
                  <div className="absolute inset-0 bg-gradient-to-t from-black/60 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
                </div>
              </div>

              <div className="space-y-4">
                <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Content Format</label>
                <select
                  value={content.content_type}
                  onChange={(e) => setContent({...content, content_type: e.target.value})}
                  className="w-full bg-white/[0.03] border border-white/10 rounded-2xl px-6 py-4 text-white font-bold focus:outline-none focus:border-red-600 transition-all appearance-none cursor-pointer"
                >
                  <option value="video" className="bg-black">🎞️ Video / Video</option>
                  <option value="vr360" className="bg-black">🥽 360° VR Experience</option>
                  <option value="image_gallery" className="bg-black">🖼️ Image Gallery</option>
                  <option value="ebook" className="bg-black">📖 E-Book</option>
                </select>
              </div>

              <div className="space-y-4">
                <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Critic Rating</label>
                <div className="flex items-center gap-4">
                  <input
                    type="range"
                    step="0.1"
                    min="0"
                    max="10"
                    value={content.rating || 0}
                    onChange={(e) => setContent({...content, rating: e.target.value})}
                    className="flex-1 accent-red-600 h-1 bg-white/10 rounded-lg appearance-none cursor-pointer"
                  />
                  <span className="text-2xl font-black text-white w-12 text-right">{parseFloat(content.rating || 0).toFixed(1)}</span>
                </div>
              </div>
            </div>
          </div>

          <div className="pt-12 border-t border-white/10 flex flex-col md:flex-row items-center justify-between gap-8">
            <div>
              {message && (
                <div className={`flex items-center gap-3 px-6 py-3 rounded-2xl ${message.includes("success") ? "bg-green-500/10 text-green-500 border border-green-500/20" : "bg-red-500/10 text-red-500 border border-red-500/20"}`}>
                  <span className="text-xl">{message.includes("success") ? "✓" : "!"}</span>
                  <p className="text-xs font-black uppercase tracking-widest">{message}</p>
                </div>
              )}
            </div>

            <div className="flex items-center gap-6">
              <Link href="/admin" className="text-white/40 hover:text-white text-xs font-black uppercase tracking-widest transition-colors">
                Discard Changes
              </Link>
              <button
                type="submit"
                disabled={saving}
                className="px-12 py-4 bg-red-600 hover:bg-red-700 disabled:bg-white/5 disabled:text-white/10 text-white rounded-2xl font-black uppercase tracking-widest transition-all shadow-[0_0_40px_rgba(220,38,38,0.2)] active:scale-95 hover:shadow-[0_0_60px_rgba(220,38,38,0.4)]"
              >
                {saving ? "Processing..." : "Commit Update"}
              </button>
            </div>
          </div>
        </form>
      </div>
    </main>
  );
}
