import { notFound } from "next/navigation";
import { api } from "@/lib/api";
import { ContentForm } from "@/components/ContentForm";

export default async function EditContentPage({
  params,
}: PageProps<"/admin/contents/[id]">) {
  const { id } = await params;
  const [allContents, discs] = await Promise.all([
    api.admin.contents.list({ limit: 1000 }).catch(() => []),
    api.admin.discs.list().catch(() => []),
  ]);
  const content = allContents.find((c) => c.id === id);
  if (!content) notFound();

  return (
    <div className="max-w-2xl space-y-4">
      <h1 className="text-xl font-bold text-gray-900">コンテンツ編集</h1>
      <ContentForm content={content} discs={discs} />
    </div>
  );
}
