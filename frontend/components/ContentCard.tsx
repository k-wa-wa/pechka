import Link from "next/link";
import Image from "next/image";
import { type MongoContent } from "@/lib/api";
import { thumbnailUrl, formatDuration } from "@/lib/utils";

const TYPE_ICONS: Record<string, string> = {
  video: "▶",
  vr360: "◎",
  image_gallery: "⊞",
  document: "☰",
};

interface Props {
  content: MongoContent;
}

export function ContentCard({ content }: Props) {
  return (
    <Link
      href={`/contents/${content.short_id}`}
      className="group block rounded-lg overflow-hidden transition-transform hover:scale-105"
      style={{ background: "var(--gh-surface)", border: "1px solid var(--gh-border)" }}
    >
      <div className="relative aspect-video" style={{ background: "#1c2128" }}>
        {content.thumbnail_key ? (
          <Image
            src={thumbnailUrl(content.thumbnail_key)}
            alt={content.title}
            fill
            className="object-cover"
            unoptimized
          />
        ) : (
          <div className="absolute inset-0 flex items-center justify-center" style={{ color: "var(--gh-muted)" }}>
            <span className="text-4xl">{TYPE_ICONS[content.content_type] ?? "▪"}</span>
          </div>
        )}
        {content.duration_seconds && (
          <span className="absolute bottom-1 right-1 bg-black/80 text-white text-xs px-1.5 py-0.5 rounded font-mono">
            {formatDuration(content.duration_seconds)}
          </span>
        )}
        {content.is_360 && (
          <span className="absolute top-1 left-1 text-xs px-1.5 py-0.5 rounded font-medium" style={{ background: "#6e40c9", color: "#fff" }}>360°</span>
        )}
      </div>
      <div className="p-3">
        <h3 className="font-medium line-clamp-2 text-sm group-hover:text-white transition-colors" style={{ color: "var(--gh-text)" }}>{content.title}</h3>
        {content.tags.length > 0 && (
          <div className="flex flex-wrap gap-1 mt-1.5">
            {content.tags.slice(0, 3).map((tag) => (
              <span key={tag} className="text-xs px-1.5 py-0.5 rounded" style={{ background: "var(--tag-bg)", color: "var(--gh-muted)" }}>
                {tag}
              </span>
            ))}
          </div>
        )}
      </div>
    </Link>
  );
}
