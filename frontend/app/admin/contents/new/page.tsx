import { api } from "@/lib/api";
import { ContentForm } from "@/components/ContentForm";

export default async function NewContentPage() {
  const discs = await api.admin.discs.list().catch(() => []);
  return (
    <div className="max-w-2xl space-y-4">
      <h1 className="text-xl font-bold text-gray-900">コンテンツ追加</h1>
      <ContentForm discs={discs} />
    </div>
  );
}
