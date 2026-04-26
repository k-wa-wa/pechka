import { Suspense } from "react";
import { api, type ContentType } from "@/lib/api";
import { ContentCard } from "@/components/ContentCard";
import { SearchBar } from "@/components/SearchBar";

const CONTENT_TYPES: { label: string; value: ContentType | "" }[] = [
  { label: "すべて", value: "" },
  { label: "動画", value: "video" },
  { label: "画像", value: "image_gallery" },
  { label: "VR", value: "vr360" },
  { label: "ドキュメント", value: "document" },
];

export default async function HomePage({
  searchParams,
}: PageProps<"/">) {
  const { type } = await searchParams;
  const contentType = type as ContentType | undefined;

  const contents = await api.contents.list({
    content_type: contentType || undefined,
    limit: 40,
  });

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row gap-4 items-start sm:items-center">
        <div className="flex-1 w-full sm:max-w-md">
          <Suspense>
            <SearchBar />
          </Suspense>
        </div>
        <div className="flex gap-2 flex-wrap">
          {CONTENT_TYPES.map((t) => (
            <a
              key={t.value}
              href={t.value ? `/?type=${t.value}` : "/"}
              className={`text-sm px-3 py-1.5 rounded-full border transition-colors ${
                (contentType ?? "") === t.value
                  ? "bg-blue-600 text-white border-blue-600"
                  : "bg-white text-gray-700 border-gray-300 hover:border-blue-400"
              }`}
            >
              {t.label}
            </a>
          ))}
        </div>
      </div>

      {contents.length === 0 ? (
        <p className="text-gray-500 text-center py-16">コンテンツがありません</p>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
          {contents.map((c) => (
            <ContentCard key={c.short_id} content={c} />
          ))}
        </div>
      )}
    </div>
  );
}
