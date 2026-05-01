import { Suspense } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { SearchBar } from "@/components/SearchBar";

const TYPE_LABELS: Record<string, string> = {
  video: "動画",
  vr360: "VR",
  image_gallery: "画像",
  document: "ドキュメント",
};

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
        <p className="text-sm" style={{ color: "var(--gh-muted)" }}>
          &quot;{query}&quot; の検索結果: {results.length} 件
        </p>
      )}

      {results.length > 0 ? (
        <ul className="space-y-2">
          {results.map((r) => (
            <li key={r.short_id}>
              <Link
                href={`/contents/${r.short_id}`}
                className="flex items-start gap-3 p-4 rounded-lg transition-colors hover:opacity-80"
                style={{ background: "var(--gh-surface)", border: "1px solid var(--gh-border)" }}
              >
                <div className="flex-1 min-w-0">
                  <h3 className="font-medium truncate" style={{ color: "var(--gh-text)" }}>{r.title}</h3>
                  {r.description && (
                    <p className="text-sm line-clamp-1 mt-0.5" style={{ color: "var(--gh-muted)" }}>{r.description}</p>
                  )}
                  {r.tags.length > 0 && (
                    <div className="flex flex-wrap gap-1 mt-1.5">
                      {r.tags.map((tag) => (
                        <span key={tag} className="text-xs px-1.5 py-0.5 rounded" style={{ background: "var(--tag-bg)", color: "var(--gh-muted)" }}>
                          {tag}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
                <span className="text-xs px-2 py-0.5 rounded shrink-0 mt-0.5" style={{ background: "var(--tag-bg)", color: "var(--gh-muted)" }}>
                  {TYPE_LABELS[r.content_type] ?? r.content_type}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      ) : query ? (
        <p className="text-center py-16" style={{ color: "var(--gh-muted)" }}>検索結果がありません</p>
      ) : (
        <p className="text-center py-16" style={{ color: "var(--gh-muted)" }}>キーワードを入力して検索</p>
      )}
    </div>
  );
}
