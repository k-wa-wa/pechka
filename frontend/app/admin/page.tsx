import { api } from "@/lib/api";
import { AdminContentTable } from "@/components/AdminContentTable";

export default async function AdminPage() {
  const [contents, discs] = await Promise.all([
    api.admin.contents.list({ limit: 100 }).catch(() => []),
    api.admin.discs.list().catch(() => []),
  ]);

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-bold" style={{ color: "var(--gh-text)" }}>管理画面</h1>

      <div className="rounded-lg overflow-hidden" style={{ border: "1px solid var(--gh-border)" }}>
        <AdminContentTable contents={contents} discs={discs} />
      </div>
    </div>
  );
}
