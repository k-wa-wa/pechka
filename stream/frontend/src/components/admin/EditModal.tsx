"use client";

import type { Content, Group, GroupPermission } from "./types";
import { GroupPermissionSelector } from "./GroupPermissionSelector";

interface Props {
  content: Content;
  groups: Group[];
  saving: boolean;
  message: { text: string; ok: boolean } | null;
  onChange: (updated: Content) => void;
  onSave: () => void;
  onClose: () => void;
}

function getThumbnail(content: Content) {
  return content.assets?.find((a) => a.asset_role === "thumbnail")?.public_url;
}

export function EditModal({ content, groups, saving, message, onChange, onSave, onClose }: Props) {
  const thumbnail = getThumbnail(content);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/80 backdrop-blur-sm" onClick={onClose} />
      <div className="relative w-full max-w-2xl bg-zinc-900 border border-white/10 rounded-3xl shadow-2xl overflow-hidden">

        {/* Header */}
        <div className="px-8 pt-8 pb-6 border-b border-white/10 flex items-start justify-between">
          <div>
            <p className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30 mb-1">Editing</p>
            <h2 className="text-2xl font-black text-white">{content.title}</h2>
            <span className="text-[10px] font-mono text-white/30">{content.short_id}</span>
          </div>
          <button onClick={onClose} className="text-white/30 hover:text-white transition-colors text-2xl leading-none mt-1 cursor-pointer">
            ×
          </button>
        </div>

        {/* Body */}
        <div className="px-8 py-6 space-y-6 max-h-[60vh] overflow-y-auto">

          {thumbnail && (
            <div className="relative aspect-video w-full rounded-xl overflow-hidden bg-white/5">
              <img src={thumbnail} alt="" className="object-cover w-full h-full" />
            </div>
          )}

          <div className="space-y-2">
            <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Title</label>
            <input
              type="text"
              value={content.title}
              onChange={(e) => onChange({ ...content, title: e.target.value })}
              className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white font-bold focus:outline-none focus:border-red-600 transition-colors"
            />
          </div>

          <div className="space-y-2">
            <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Description</label>
            <textarea
              value={content.description}
              onChange={(e) => onChange({ ...content, description: e.target.value })}
              rows={4}
              className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white font-medium focus:outline-none focus:border-red-600 transition-colors resize-none"
            />
          </div>

          {/* Access Control */}
          <div className="space-y-3">
            <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Access Control</label>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={() => onChange({ ...content, visibility: "public", group_permissions: [] })}
                className={`flex-1 py-2.5 rounded-xl text-xs font-black uppercase tracking-widest border transition-all ${
                  content.visibility === "public"
                    ? "bg-green-600/20 border-green-500 text-green-400"
                    : "bg-white/5 border-white/10 text-white/30 hover:text-white"
                }`}
              >
                Public
              </button>
              <button
                type="button"
                onClick={() => onChange({ ...content, visibility: "group_only" })}
                className={`flex-1 py-2.5 rounded-xl text-xs font-black uppercase tracking-widest border transition-all ${
                  content.visibility === "group_only"
                    ? "bg-yellow-600/20 border-yellow-500 text-yellow-400"
                    : "bg-white/5 border-white/10 text-white/30 hover:text-white"
                }`}
              >
                Group Only
              </button>
            </div>

            {content.visibility === "group_only" && (
              <div className="space-y-2">
                <p className="text-[10px] text-white/30">
                  グループを追加し R(読取) / W(編集) / D(削除) 権限を設定
                </p>
                <GroupPermissionSelector
                  groups={groups}
                  value={content.group_permissions || []}
                  onChange={(perms: GroupPermission[]) =>
                    onChange({ ...content, group_permissions: perms })
                  }
                />
              </div>
            )}
          </div>

          {/* Assets */}
          <div className="space-y-4">
            <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Assets</label>
            <div className="space-y-3">
              {(content.assets || []).length > 0 ? (
                content.assets.map((asset, idx) => (
                  <div key={idx} className="bg-white/5 border border-white/10 rounded-xl p-4 flex flex-col gap-2">
                    <div className="flex justify-between items-center">
                      <span className="text-[10px] font-bold uppercase text-red-500 bg-red-500/10 px-2 py-0.5 rounded-full">
                        {asset.asset_role}
                      </span>
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
                <div className="text-sm text-white/20 italic p-4 border border-dashed border-white/10 rounded-xl">
                  No assets attached.
                </div>
              )}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-6">
            <div className="space-y-2">
              <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Type</label>
              <div className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white/50 font-bold">
                {content.content_type} (Read Only)
              </div>
            </div>
            <div className="space-y-2">
              <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">
                Rating: <span className="text-white">{parseFloat(String(content.rating ?? 0)).toFixed(1)}</span>
              </label>
              <input
                type="range"
                step="0.1"
                min="0"
                max="10"
                value={content.rating ?? 0}
                onChange={(e) => onChange({ ...content, rating: parseFloat(e.target.value) })}
                className="w-full accent-red-600"
              />
            </div>
          </div>

          <div className="space-y-2">
            <label className="text-[10px] font-black uppercase tracking-[0.2em] text-white/30">Tags (Comma separated)</label>
            <input
              type="text"
              value={(content.tags || []).join(", ")}
              onChange={(e) =>
                onChange({ ...content, tags: e.target.value.split(",").map((t) => t.trim()).filter((t) => t !== "") })
              }
              placeholder="Action, Sci-Fi, Adventure"
              className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white font-medium focus:outline-none focus:border-red-600 transition-colors"
            />
          </div>
        </div>

        {/* Footer */}
        <div className="px-8 py-6 border-t border-white/10 flex items-center justify-between">
          <div>
            {message && (
              <p className={`text-xs font-black uppercase tracking-wider ${message.ok ? "text-green-500" : "text-red-500"}`}>
                {message.ok ? "✓ " : "! "}{message.text}
              </p>
            )}
          </div>
          <div className="flex gap-4">
            <button
              onClick={onClose}
              className="text-white/30 hover:text-white text-xs font-black uppercase tracking-widest transition-colors cursor-pointer"
            >
              Cancel
            </button>
            <button
              onClick={onSave}
              disabled={saving}
              className="px-8 py-3 bg-red-600 hover:bg-red-700 disabled:opacity-30 text-white rounded-xl font-black uppercase tracking-widest transition-all active:scale-95 cursor-pointer"
            >
              {saving ? "Saving..." : "Save Changes"}
            </button>
          </div>
        </div>

      </div>
    </div>
  );
}
