"use client";

import { useState, useEffect, useCallback } from "react";
import { apiClient } from "@/lib/api-client";
import type { Content, Group } from "@/components/admin/types";
import { ContentTypeTabs } from "@/components/admin/ContentTypeTabs";
import { ContentTable } from "@/components/admin/ContentTable";
import { EditModal } from "@/components/admin/EditModal";

const API_BASE = "/api/metadata/v1";

export default function AdminDashboard() {
  const [contents, setContents] = useState<Content[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(true);
  const [editTarget, setEditTarget] = useState<Content | null>(null);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ text: string; ok: boolean } | null>(null);

  const [isBulkEdit, setIsBulkEdit] = useState(false);
  const [bulkChanges, setBulkChanges] = useState<Record<string, Partial<Content>>>({});
  const [bulkSyncing, setBulkSyncing] = useState(false);

  const [activeTab, setActiveTab] = useState<Content["content_type"]>("video");

  const fetchContents = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiClient.get(`${API_BASE}/admin/metadata/contents`);
      setContents(
        (res.data || []).sort(
          (a: Content, b: Content) =>
            new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        )
      );
    } catch (e) {
      console.error("Failed to fetch:", e);
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchGroups = useCallback(async () => {
    try {
      const res = await apiClient.get(`${API_BASE}/admin/metadata/groups`);
      setGroups(res.data || []);
    } catch (e) {
      console.error("Failed to fetch groups:", e);
    }
  }, []);

  useEffect(() => {
    fetchContents();
    fetchGroups();
  }, [fetchContents, fetchGroups]);

  const openModal = (content: Content) => {
    setEditTarget({
      ...content,
      visibility: content.visibility || "public",
      allowed_groups: content.allowed_groups || [],
    });
    setMessage(null);
  };

  const closeModal = () => {
    setEditTarget(null);
    setMessage(null);
  };

  const handleSave = async () => {
    if (!editTarget) return;
    setSaving(true);
    setMessage(null);
    try {
      await apiClient.put(`${API_BASE}/admin/metadata/contents/${editTarget.id}`, {
        title: editTarget.title,
        description: editTarget.description,
        rating: editTarget.rating ? parseFloat(String(editTarget.rating)) : 0,
        tags: editTarget.tags || [],
        content_type: editTarget.content_type,
        visibility: editTarget.visibility,
        allowed_groups: editTarget.allowed_groups || [],
      });
      setMessage({ text: "保存しました！", ok: true });
      await fetchContents();
      setTimeout(closeModal, 1200);
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ||
        "保存に失敗しました";
      setMessage({ text: msg, ok: false });
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
        const content = contents.find((c) => c.id === id);
        if (!content) continue;
        const changes = bulkChanges[id];
        await apiClient.put(`${API_BASE}/admin/metadata/contents/${id}`, {
          ...content,
          ...changes,
          rating: changes.rating ?? content.rating ?? 0,
        });
        successCount++;
      }
      setMessage({ text: `${successCount}件の変更を保存しました`, ok: true });
      setBulkChanges({});
      setIsBulkEdit(false);
      await fetchContents();
    } catch {
      setMessage({ text: "一括保存中にエラーが発生しました", ok: false });
    } finally {
      setSaving(false);
    }
  };

  const handleSync = async (_id: string, shortId: string) => {
    setBulkSyncing(true);
    try {
      await apiClient.post(`/api/catalog/v1/internal/catalog/sync/${shortId}`);
      setMessage({ text: "カタログに同期しました", ok: true });
    } catch {
      setMessage({ text: "同期に失敗しました", ok: false });
    } finally {
      setBulkSyncing(false);
    }
  };

  const updateBulkChange = (id: string, field: string, value: unknown) => {
    setBulkChanges((prev) => ({
      ...prev,
      [id]: { ...prev[id], [field]: value },
    }));
  };

  const filteredContents = contents.filter((c) => c.content_type === activeTab);

  return (
    <main className="min-h-screen bg-black pt-24 px-4 md:px-12 pb-24">
      <header className="mb-12 flex justify-between items-end">
        <div>
          <h1 className="text-4xl md:text-5xl font-black tracking-tight mb-2 text-white">Admin Dashboard</h1>
          <p className="text-white/50 border-l-2 border-red-600 pl-4">Manage unified content metadata</p>
        </div>
        <div className="flex gap-4">
          <button
            onClick={() => {
              if (isBulkEdit) handleBulkSave();
              else setIsBulkEdit(true);
            }}
            className={`px-6 py-2 rounded-full transition-all text-sm font-bold border ${
              isBulkEdit
                ? "bg-green-600 border-green-500 text-white shadow-lg shadow-green-600/20"
                : "bg-white/10 border-white/10 text-white hover:bg-white/20"
            }`}
          >
            {isBulkEdit ? (saving ? "Saving Changes..." : "Save Bulk Changes") : "Edit All Data"}
          </button>
        </div>
      </header>

      <ContentTypeTabs activeTab={activeTab} onChange={setActiveTab} />

      <ContentTable
        contents={filteredContents}
        loading={loading}
        isBulkEdit={isBulkEdit}
        bulkChanges={bulkChanges}
        bulkSyncing={bulkSyncing}
        onEdit={openModal}
        onSync={handleSync}
        onBulkChange={updateBulkChange}
      />

      {editTarget && (
        <EditModal
          content={editTarget}
          groups={groups}
          saving={saving}
          message={message}
          onChange={setEditTarget}
          onSave={handleSave}
          onClose={closeModal}
        />
      )}
    </main>
  );
}
