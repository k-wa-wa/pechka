import { Suspense } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { SearchBar } from "@/components/SearchBar";

export default async function SearchPage({
  searchParams,
}: PageProps<"/search">) {
  const { q } = await searchParams;
  const query = typeof q === "string" ? q : "";

  const results = query ? await api.search(query, { limit: 40 }).catch(() => []) : [];

  return (
    <div className="space-y-6">
      <div className="max-w-md">
        <Suspense>
          <SearchBar />
        </Suspense>
      </div>

      {query && (
        <p className="text-sm text-gray-500">
          &quot;{query}&quot; の検索結果: {results.length} 件
        </p>
      )}

      {results.length > 0 ? (
        <ul className="space-y-3">
          {results.map((r) => (
            <li key={r.short_id}>
              <Link
                href={`/contents/${r.short_id}`}
                className="block p-4 bg-white border border-gray-200 rounded-lg hover:border-blue-400 transition-colors"
              >
                <div className="flex items-start gap-3">
                  <div className="flex-1 min-w-0">
                    <h3 className="font-medium text-gray-900 truncate">{r.title}</h3>
                    {r.description && (
                      <p className="text-sm text-gray-600 line-clamp-2 mt-0.5">{r.description}</p>
                    )}
                    {r.tags.length > 0 && (
                      <div className="flex flex-wrap gap-1 mt-1">
                        {r.tags.map((tag) => (
                          <span key={tag} className="text-xs bg-gray-100 text-gray-600 px-1.5 py-0.5 rounded">
                            {tag}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                  <span className="text-xs text-gray-500 shrink-0">{r.content_type}</span>
                </div>
              </Link>
            </li>
          ))}
        </ul>
      ) : query ? (
        <p className="text-gray-500 text-center py-16">検索結果がありません</p>
      ) : (
        <p className="text-gray-500 text-center py-16">キーワードを入力して検索</p>
      )}
    </div>
  );
}
