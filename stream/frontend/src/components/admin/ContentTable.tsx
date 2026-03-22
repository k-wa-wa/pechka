"use client";

import type { Content } from "./types";

interface BulkChanges {
  [id: string]: Partial<Content>;
}

interface Props {
  contents: Content[];
  loading: boolean;
  isBulkEdit: boolean;
  bulkChanges: BulkChanges;
  bulkSyncing: boolean;
  onEdit: (content: Content) => void;
  onSync: (id: string, shortId: string) => void;
  onBulkChange: (id: string, field: string, value: unknown) => void;
}

function getThumbnail(content: Content) {
  return content.assets?.find((a) => a.asset_role === "thumbnail")?.public_url;
}

function VisibilityBadge({ visibility }: { visibility: string }) {
  return (
    <span
      className={`text-[9px] font-bold uppercase px-1.5 py-0.5 rounded-full ${
        visibility === "public"
          ? "bg-green-600/20 text-green-400"
          : "bg-yellow-600/20 text-yellow-400"
      }`}
    >
      {visibility === "public" ? "Public" : "Group Only"}
    </span>
  );
}

export function ContentTable({
  contents,
  loading,
  isBulkEdit,
  bulkChanges,
  bulkSyncing,
  onEdit,
  onSync,
  onBulkChange,
}: Props) {
  return (
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
              <tr>
                <td colSpan={5} className="px-6 py-20 text-center text-white/20 animate-pulse">
                  Loading...
                </td>
              </tr>
            ) : contents.length > 0 ? (
              contents.map((content) => (
                <ContentRow
                  key={content.id}
                  content={content}
                  isBulkEdit={isBulkEdit}
                  bulkChange={bulkChanges[content.id]}
                  bulkSyncing={bulkSyncing}
                  onEdit={onEdit}
                  onSync={onSync}
                  onBulkChange={onBulkChange}
                />
              ))
            ) : (
              <tr>
                <td colSpan={5} className="px-6 py-20 text-center text-white/20 italic">
                  No content found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

interface RowProps {
  content: Content;
  isBulkEdit: boolean;
  bulkChange: Partial<Content> | undefined;
  bulkSyncing: boolean;
  onEdit: (content: Content) => void;
  onSync: (id: string, shortId: string) => void;
  onBulkChange: (id: string, field: string, value: unknown) => void;
}

function ContentRow({ content, isBulkEdit, bulkChange, bulkSyncing, onEdit, onSync, onBulkChange }: RowProps) {
  const thumbnail = getThumbnail(content);

  return (
    <tr className="hover:bg-white/[0.02] transition-colors group">
      <td className="px-6 py-4">
        <div className="relative w-24 aspect-video rounded-md overflow-hidden bg-white/10 border border-white/10">
          {thumbnail ? (
            <img src={thumbnail} alt={content.title} className="object-cover w-full h-full" />
          ) : (
            <div className="w-full h-full flex items-center justify-center text-[10px] text-white/20">No Image</div>
          )}
        </div>
      </td>

      <td className="px-6 py-4">
        {isBulkEdit ? (
          <input
            type="text"
            value={bulkChange?.title ?? content.title}
            onChange={(e) => onBulkChange(content.id, "title", e.target.value)}
            className={`w-full bg-white/10 border rounded px-2 py-1 text-white font-bold focus:outline-none transition-colors ${
              bulkChange?.title !== undefined
                ? "border-green-500 shadow-[0_0_10px_rgba(34,197,94,0.3)]"
                : "border-white/20 focus:border-red-600"
            }`}
          />
        ) : (
          <>
            <div className="font-bold text-white group-hover:text-red-500 transition-colors">{content.title}</div>
            <div className="text-[10px] text-white/40 font-mono mt-1">ID: {content.short_id}</div>
            <div className="mt-1">
              <VisibilityBadge visibility={content.visibility} />
            </div>
          </>
        )}
      </td>

      <td className="px-6 py-4">
        {isBulkEdit ? (
          <textarea
            value={bulkChange?.description ?? content.description}
            onChange={(e) => onBulkChange(content.id, "description", e.target.value)}
            rows={2}
            className={`w-full bg-white/10 border rounded px-2 py-1 text-white text-xs focus:outline-none resize-none transition-colors ${
              bulkChange?.description !== undefined
                ? "border-green-500 shadow-[0_0_10px_rgba(34,197,94,0.3)]"
                : "border-white/20 focus:border-red-600"
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
              onClick={() => onEdit(content)}
              title="Edit"
              className="p-2 bg-white/5 hover:bg-white/10 text-white rounded-lg transition-colors border border-white/10 cursor-pointer"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
              </svg>
            </button>
            <button
              onClick={() => onSync(content.id, content.short_id)}
              disabled={bulkSyncing}
              title="Sync to Catalog"
              className="p-2 bg-white/5 hover:bg-blue-600/10 text-blue-500 rounded-lg transition-colors border border-white/10 cursor-pointer disabled:opacity-30"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            </button>
            <button
              title="Delete (Coming Soon)"
              className="p-2 bg-red-600/5 hover:bg-red-600/10 text-red-500/30 rounded-lg transition-colors border border-red-600/10 cursor-not-allowed"
              disabled
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        )}
      </td>
    </tr>
  );
}
