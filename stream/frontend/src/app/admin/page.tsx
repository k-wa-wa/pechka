"use client";

import { useState, useEffect, useCallback } from "react";
import Link from "next/link";

const API_BASE = "/api/metadata/v1";

interface Content {
  id: string;
  short_id: string;
  type: "video" | "gallery" | "vr360";
  title: string;
  description: string;
  rating: number | null;
  created_at: string;
  tags: string[];
  assets: Array<{ asset_role: string; public_url: string }>;
}

export default function AdminDashboard() {
  const [contents, setContents] = useState<Content[]>([]);
  const [loading, setLoading] = useState(true);
  const [editTarget, setEditTarget] = useState<Content | null>(null);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ text: string; ok: boolean } | null>(null);

  // Bulk Edit & Selection States
  // Bulk Edit States
  const [isBulkEdit, setIsBulkEdit] = useState(false);
  const [bulkChanges, setBulkChanges] = useState<Record<string, Partial<Content>>>({});
  const [bulkSyncing, setBulkSyncing] = useState(false);

  // Tab State for Segmented Control
  const [activeTab, setActiveTab] = useState<"video" | "vr360" | "gallery">("video");

  const fetchContents = useCallback(async () => {
    setLoading(true);
    try {
      // Fetch both videos and galleries
      const [vRes, gRes] = await Promise.all([
        fetch(`${API_BASE}/admin/metadata/videos`),
        fetch(`${API_BASE}/admin/metadata/galleries`),
      ]);

      let allData: Content[] = [];
      if (vRes.ok) {
        const videos = await vRes.json();
        allData = [...allData, ...(videos || []).map((v: any) => ({ ...v, type: v.is_360 ? "vr360" : "video" }))];
      }
      if (gRes.ok) {
        const galleries = await gRes.json();
        allData = [...allData, ...(galleries || []).map((g: any) => ({ ...g, type: "gallery" }))];
      }

      setContents(allData.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()));
    } catch (e) {
      console.error("Failed to fetch:", e);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchContents(); }, [fetchContents]);

  const openModal = (content: Content) => {
    setEditTarget({ ...content });
    setMessage(null);
  };
  const closeModal = () => { setEditTarget(null); setMessage(null); };

  const handleSave = async () => {
    if (!editTarget) return;
    setSaving(true);
    setMessage(null);
    try {
      const endpoint = (editTarget.type === "video" || editTarget.type === "vr360") 
        ? `${API_BASE}/admin/metadata/videos/${editTarget.id}`
        : `${API_BASE}/admin/metadata/galleries/${editTarget.id}`;

      const res = await fetch(endpoint, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          title: editTarget.title,
          description: editTarget.description,
          rating: editTarget.rating ? parseFloat(String(editTarget.rating)) : 0,
          tags: editTarget.tags || [],
        }),
      });
      if (res.ok) {
        setMessage({ text: "保存しました！", ok: true });
        await fetchContents();
        setTimeout(closeModal, 1200);
      } else {
        const err = await res.json();
        setMessage({ text: err.error || "保存に失敗しました", ok: false });
      }
    } catch {
      setMessage({ text: "通信エラーが発生しました", ok: false });
    } finally {
      setSaving(false);
    }
  };

  const handleBulkSave = async () => {
    const idsToSave = Object.keys(bulkChanges);
    if (idsToSave.length === 0) {
      setIsBulkEdit(false);
      return;
    }

    setSaving(true);
    let successCount = 0;
    
    try {
      for (const id of idsToSave) {
        const content = contents.find(c => c.id === id);
        if (!content) continue;
        const changes = bulkChanges[id];
        
        const endpoint = (content.type === "video" || content.type === "vr360") 
          ? `${API_BASE}/admin/metadata/videos/${id}`
          : `${API_BASE}/admin/metadata/galleries/${id}`;

        const res = await fetch(endpoint, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            ...content,
            ...changes,
            rating: changes.rating ?? content.rating ?? 0,
          }),
        });
        if (res.ok) successCount++;
      }
      
      setMessage({ text: `${successCount}件の変更を保存しました`, ok: true });
      setBulkChanges({});
      setIsBulkEdit(false);
      await fetchContents();
    } catch (e) {
      setMessage({ text: "一括保存中にエラーが発生しました", ok: false });
    } finally {
      setSaving(false);
    }
  };

  const handleBulkSync = async (id: string, shortId: string) => {
    setBulkSyncing(true);
    try {
      const res = await fetch(`/api/catalog/v1/internal/catalog/sync/${shortId}`, { method: "POST" });
      if (res.ok) {
        setMessage({ text: "カタログに同期しました", ok: true });
      } else {
        setMessage({ text: "同期に失敗しました", ok: false });
      }
    } catch (e) {
      setMessage({ text: "同期中にエラーが発生しました", ok: false });
    } finally {
      setBulkSyncing(false);
    }
  };

  const toggleSelectAll = () => {}; // Removed
  const toggleSelect = (id: string) => {}; // Removed

  const updateBulkChange = (id: string, field: string, value: any) => {
    setBulkChanges(prev => ({
      ...prev,
      [id]: { ...prev[id], [field]: value }
    }));
  };

  const filteredContents = contents.filter(c => c.type === activeTab);

  const getThumbnail = (content: Content) =>
    content.assets?.find(a => a.asset_role === "thumbnail")?.public_url;

  return (
    <main className="min-h-screen bg-black pt-24 px-4 md:px-12 pb-24">
      <header className="mb-12 flex justify-between items-end">
        <div>
          <h1 className="text-4xl md:text-5xl font-black tracking-tight mb-2 text-white">Admin Dashboard</h1>
          <p className="text-white/50 border-l-2 border-red-600 pl-4">Manage separated domain content (Videos & Galleries)</p>
        </div>
        <div className="flex gap-4">
          <button
            onClick={() => {
              if (isBulkEdit) handleBulkSave();
              else setIsBulkEdit(true);
            }}
            className={`px-6 py-2 rounded-full transition-all text-sm font-bold border ${
              isBulkEdit ? "bg-green-600 border-green-500 text-white shadow-lg shadow-green-600/20" : "bg-white/10 border-white/10 text-white hover:bg-white/20"
            }`}
          >
            {isBulkEdit ? (saving ? "Saving Changes..." : "Save Bulk Changes") : "Edit All Data"}
          </button>
        </div>
      </header>

      {/* Segmented Control */}
      <div className="flex p-1 bg-zinc-900 border border-white/10 rounded-xl mb-8 w-fit mx-auto">
        {(["video", "vr360", "gallery"] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => {
              setActiveTab(tab);
            }}
            className={`px-8 py-2 rounded-lg text-xs font-black uppercase tracking-widest transition-all ${
              activeTab === tab ? "bg-white text-black" : "text-white/40 hover:text-white"
            }`}
          >
            {tab === "vr360" ? "360° VR" : tab === "video" ? "Videos" : "Gallery"}
          </button>
        ))}
      </div>

      <div className="bg-zinc-900/50 rounded-2xl border border-white/10 overflow-hidden backdrop-blur-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-white/10 bg-white/5">
                <th className="px-6 py-4 text-xs font-bold uppercase tracking-wider text-white/40">Thumbnail</th>
                <th className="px-6 py-4 text-xs font-bold uppercase tracking-wider text-white/40">Title & ID</th>
                <th className="px-6 py-4 text-xs font-bold uppercase tracking-wider text-white/40">Description</th>
                <th className="px-6 py-4 text-xs font-bold uppercase tracking-wider text-white/40">Created At</th>
                <th className="px-6 py-4 text-xs font-bold uppercase tracking-wider text-white/40 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {loading ? (
                <tr><td colSpan={6} className="px-6 py-20 text-center text-white/20 animate-pulse">Loading...</td></tr>
              ) : filteredContents.length > 0 ? (
                filteredContents.map((content) => (
                  <tr key={content.id} className="hover:bg-white/[0.02] transition-colors group">
                    <td className="px-6 py-4">
                      <div className="relative w-24 aspect-video rounded-md overflow-hidden bg-white/10 border border-white/10">
                        {getThumbnail(content) ? (
                          <img src={getThumbnail(content)} alt={content.title} className="object-cover w-full h-full" />
                        ) : (
                          <div className="w-full h-full flex items-center justify-center text-[10px] text-white/20">No Image</div>
                        )}
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      {isBulkEdit ? (
                        <input
                          type="text"
                          value={bulkChanges[content.id]?.title ?? content.title}
                          onChange={(e) => updateBulkChange(content.id, "title", e.target.value)}
                          className={`w-full bg-white/10 border rounded px-2 py-1 text-white font-bold focus:outline-none transition-colors ${
                            bulkChanges[content.id]?.title !== undefined ? "border-green-500 shadow-[0_0_10px_rgba(34,197,94,0.3)]" : "border-white/20 focus:border-red-600"
                          }`}
                        />
                      ) : (
                        <>
                          <div className="font-bold text-white group-hover:text-red-500 transition-colors">{content.title}</div>
                          <div className="text-[10px] text-white/40 font-mono mt-1">ID: {content.short_id}</div>
                        </>
                      )}
                    </td>
                    <td className="px-6 py-4">
                      {isBulkEdit ? (
                        <textarea
                          value={bulkChanges[content.id]?.description ?? content.description}
                          onChange={(e) => updateBulkChange(content.id, "description", e.target.value)}
                          rows={2}
                          className={`w-full bg-white/10 border rounded px-2 py-1 text-white text-xs focus:outline-none resize-none transition-colors ${
                            bulkChanges[content.id]?.description !== undefined ? "border-green-500 shadow-[0_0_10px_rgba(34,197,94,0.3)]" : "border-white/20 focus:border-red-600"
                          }`}
                        />
                      ) : (
                        <div className="text-xs text-white/40 line-clamp-2 max-w-sm">{content.description}</div>
                      )}
                    </td>
                    <td className="px-6 py-4 text-sm text-white/50">
                      {content.created_at ? new Date(content.created_at).toLocaleDateString("ja-JP") : "N/A"}
                    </td>
                    <td className="px-6 py-4 text-right">
                      {!isBulkEdit && (
                        <div className="flex justify-end gap-2">
                          <button
                            onClick={() => openModal(content)}
                            title="Edit"
                            className="p-2 bg-white/5 hover:bg-white/10 text-white rounded-lg transition-colors border border-white/10 cursor-pointer"
                          >
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" /></svg>
                          </button>
                          <button
                            onClick={() => handleBulkSync(content.id, content.short_id)}
                            disabled={bulkSyncing}
                            title="Sync to Catalog"
                            className="p-2 bg-white/5 hover:bg-blue-600/10 text-blue-500 rounded-lg transition-colors border border-white/10 cursor-pointer disabled:opacity-30"
                          >
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
                          </button>
                          <button
                            title="Delete (Coming Soon)"
                            className="p-2 bg-red-600/5 hover:bg-red-600/10 text-red-500/30 rounded-lg transition-colors border border-red-600/10 cursor-not-allowed"
                            disabled
                          >
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
                          </button>
                        </div>
                      )}
                    </td>
                  </tr>
                ))
              ) : (
                <tr><td colSpan={6} className="px-6 py-20 text-center text-white/20 italic">No content found.</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Bulk Actions Bar Removed */}

      {/* Edit Modal */}
      {editTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/80 backdrop-blur-sm" onClick={closeModal} />
          <div className="relative w-full max-w-2xl bg-zinc-900 border border-white/10 rounded-3xl shadow-2xl overflow-hidden">
            
            {/* Modal Header */}
            <div className="px-8 pt-8 pb-6 border-b border-white/10 flex items-start justify-between">
              <div>
                <p className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30 mb-1">Editing</p>
                <h2 className="text-2xl font-black text-white">{editTarget.title}</h2>
                <span className="text-[10px] font-mono text-white/30">{editTarget.short_id}</span>
              </div>
              <button onClick={closeModal} className="text-white/30 hover:text-white transition-colors text-2xl leading-none mt-1 cursor-pointer">×</button>
            </div>

            {/* Modal Body */}
            <div className="px-8 py-6 space-y-6 max-h-[60vh] overflow-y-auto">
              
              {/* Thumbnail preview */}
              {getThumbnail(editTarget) && (
                <div className="relative aspect-video w-full rounded-xl overflow-hidden bg-white/5">
                  <img src={getThumbnail(editTarget)} alt="" className="object-cover w-full h-full" />
                </div>
              )}

              <div className="space-y-2">
                <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Title</label>
                <input
                  type="text"
                  value={editTarget.title}
                  onChange={(e) => setEditTarget({ ...editTarget, title: e.target.value })}
                  className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white font-bold focus:outline-none focus:border-red-600 transition-colors"
                />
              </div>

              <div className="space-y-2">
                <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Description</label>
                <textarea
                  value={editTarget.description}
                  onChange={(e) => setEditTarget({ ...editTarget, description: e.target.value })}
                  rows={4}
                  className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white font-medium focus:outline-none focus:border-red-600 transition-colors resize-none"
                />
              </div>

              <div className="space-y-4">
                <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Assets</label>
                <div className="space-y-3">
                  {(editTarget.assets || []).length > 0 ? (
                    editTarget.assets.map((asset, idx) => (
                      <div key={idx} className="bg-white/5 border border-white/10 rounded-xl p-4 flex flex-col gap-2">
                        <div className="flex justify-between items-center">
                          <span className="text-[10px] font-bold uppercase text-red-500 bg-red-500/10 px-2 py-0.5 rounded-full">{asset.asset_role}</span>
                          <span className="text-[10px] font-mono text-white/30 truncate max-w-[200px]">{asset.public_url}</span>
                        </div>
                        {asset.asset_role === "thumbnail" && (
                          <div className="relative aspect-video w-32 rounded-lg overflow-hidden border border-white/10 mt-1">
                            <img src={asset.public_url} alt="Thumbnail preview" className="object-cover w-full h-full" />
                          </div>
                        )}
                      </div>
                    ))
                  ) : (
                    <div className="text-sm text-white/20 italic p-4 border border-dashed border-white/10 rounded-xl">No assets attached.</div>
                  )}
                </div>
              </div>

              <div className="grid grid-cols-2 gap-6">
                <div className="space-y-2">
                  <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Type</label>
                  <div className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white/50 font-bold">
                    {editTarget.type === "video" ? "🎞️ Video / Video" : editTarget.type === "vr360" ? "🥽 360° VR" : "🖼️ Gallery"} (Read Only)
                  </div>
                </div>
                <div className="space-y-2">
                  <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">
                    Rating: <span className="text-white">{parseFloat(String(editTarget.rating ?? 0)).toFixed(1)}</span>
                  </label>
                  <input
                    type="range"
                    step="0.1"
                    min="0"
                    max="10"
                  />
                </div>
              </div>

              <div className="space-y-2">
                <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Tags (Comma separated)</label>
                <input
                  type="text"
                  value={(editTarget.tags || []).join(", ")}
                  onChange={(e) => setEditTarget({ ...editTarget, tags: e.target.value.split(",").map(t => t.trim()).filter(t => t !== "") })}
                  placeholder="Action, Sci-Fi, Adventure"
                  className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white font-medium focus:outline-none focus:border-red-600 transition-colors"
                />
              </div>
            </div>

            {/* Modal Footer */}
            <div className="px-8 py-6 border-t border-white/10 flex items-center justify-between">
              <div>
                {message && (
                  <p className={`text-xs font-black uppercase tracking-wider ${message.ok ? "text-green-500" : "text-red-500"}`}>
                    {message.ok ? "✓ " : "! "}{message.text}
                  </p>
                )}
              </div>
              <div className="flex gap-4">
                <button onClick={closeModal} className="text-white/30 hover:text-white text-xs font-black uppercase tracking-widest transition-colors cursor-pointer">
                  Cancel
                </button>
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="px-8 py-3 bg-red-600 hover:bg-red-700 disabled:opacity-30 text-white rounded-xl font-black uppercase tracking-widest transition-all active:scale-95 cursor-pointer"
                >
                  {saving ? "Saving..." : "Save Changes"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </main>
  );
}
