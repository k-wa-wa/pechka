"use client";

import { useState, useRef, useEffect } from "react";
import type { Group, GroupPermission } from "./types";

interface Props {
  groups: Group[];
  value: GroupPermission[];
  onChange: (perms: GroupPermission[]) => void;
}

const PERM_LABELS = [
  { key: "can_read",   label: "R", title: "Read" },
  { key: "can_write",  label: "W", title: "Write" },
  { key: "can_delete", label: "D", title: "Delete" },
] as const;

export function GroupPermissionSelector({ groups, value, onChange }: Props) {
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Close dropdown on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  const selectedIds = new Set(value.map((p) => p.group_id));

  const filtered = groups.filter(
    (g) => !selectedIds.has(g.id) && g.name.toLowerCase().includes(query.toLowerCase())
  );

  function addGroup(group: Group) {
    onChange([...value, { group_id: group.id, can_read: true, can_write: false, can_delete: false }]);
    setQuery("");
    setOpen(false);
  }

  function removeGroup(groupId: string) {
    onChange(value.filter((p) => p.group_id !== groupId));
  }

  function togglePerm(groupId: string, key: "can_read" | "can_write" | "can_delete") {
    onChange(
      value.map((p) =>
        p.group_id === groupId ? { ...p, [key]: !p[key] } : p
      )
    );
  }

  function getGroupName(id: string) {
    return groups.find((g) => g.id === id)?.name ?? id;
  }

  return (
    <div className="space-y-3">
      {/* Searchable input */}
      <div ref={containerRef} className="relative">
        <input
          type="text"
          value={query}
          onChange={(e) => { setQuery(e.target.value); setOpen(true); }}
          onFocus={() => setOpen(true)}
          placeholder="グループを検索して追加..."
          className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-2.5 text-sm text-white placeholder:text-white/20 focus:outline-none focus:border-white/30 transition-colors"
        />
        {open && filtered.length > 0 && (
          <ul className="absolute z-20 mt-1 w-full max-h-48 overflow-y-auto bg-zinc-800 border border-white/10 rounded-xl shadow-2xl">
            {filtered.map((g) => (
              <li key={g.id}>
                <button
                  type="button"
                  onClick={() => addGroup(g)}
                  className="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors"
                >
                  {g.name}
                  {g.description && (
                    <span className="ml-2 text-[10px] text-white/30">{g.description}</span>
                  )}
                </button>
              </li>
            ))}
          </ul>
        )}
        {open && query && filtered.length === 0 && (
          <div className="absolute z-20 mt-1 w-full bg-zinc-800 border border-white/10 rounded-xl px-4 py-3 text-xs text-white/30">
            一致するグループが見つかりません
          </div>
        )}
      </div>

      {/* Selected groups with permission toggles */}
      {value.length > 0 && (
        <ul className="space-y-2">
          {value.map((perm) => (
            <li
              key={perm.group_id}
              className="flex items-center justify-between gap-3 bg-white/5 border border-white/10 rounded-xl px-4 py-2.5"
            >
              <span className="text-sm font-bold text-white truncate flex-1">
                {getGroupName(perm.group_id)}
              </span>

              <div className="flex items-center gap-1.5 flex-shrink-0">
                {PERM_LABELS.map(({ key, label, title }) => (
                  <button
                    key={key}
                    type="button"
                    title={title}
                    onClick={() => togglePerm(perm.group_id, key)}
                    className={`w-7 h-7 rounded-lg text-[11px] font-black border transition-all ${
                      perm[key]
                        ? key === "can_read"
                          ? "bg-blue-600/30 border-blue-500 text-blue-400"
                          : key === "can_write"
                          ? "bg-yellow-600/30 border-yellow-500 text-yellow-400"
                          : "bg-red-600/30 border-red-500 text-red-400"
                        : "bg-white/5 border-white/10 text-white/20 hover:text-white/50"
                    }`}
                  >
                    {label}
                  </button>
                ))}
                <button
                  type="button"
                  onClick={() => removeGroup(perm.group_id)}
                  className="w-7 h-7 rounded-lg text-white/20 hover:text-white hover:bg-white/10 border border-transparent transition-all text-sm flex items-center justify-center"
                >
                  ×
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}

      {value.length === 0 && (
        <p className="text-xs text-white/20 italic">グループが選択されていません</p>
      )}
    </div>
  );
}
