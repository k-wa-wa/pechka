import Link from "next/link";
import { api } from "@/lib/api";
import { AdminContentTable } from "@/components/AdminContentTable";

export default async function AdminPage() {
  const [contents, discs] = await Promise.all([
    api.admin.contents.list({ limit: 100 }).catch(() => []),
    api.admin.discs.list().catch(() => []),
  ]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold text-gray-900">管理画面</h1>
        <Link
          href="/admin/contents/new"
          className="bg-blue-600 text-white px-4 py-2 rounded-lg text-sm hover:bg-blue-700 transition-colors"
        >
          + コンテンツ追加
        </Link>
      </div>

      <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <AdminContentTable contents={contents} discs={discs} />
      </div>
    </div>
  );
}
