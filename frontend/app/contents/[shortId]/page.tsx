import { notFound } from "next/navigation";
import { Suspense } from "react";
import { api } from "@/lib/api";
import { HlsPlayer } from "@/components/HlsPlayer";
import { VrViewer } from "@/components/VrViewer";
import { thumbnailUrl, formatDuration } from "@/lib/utils";
import Image from "next/image";

const TYPE_ICONS: Record<string, string> = {
  video: "▶",
  vr360: "◎",
  image_gallery: "⊞",
  document: "☰",
};

const STATUS_STYLE: Record<string, { bg: string; text: string }> = {
  ready: { bg: "#1a3a2a", text: "#3fb950" },
  processing: { bg: "#332900", text: "#d29922" },
  error: { bg: "#3d0d0d", text: "#f85149" },
  pending: { bg: "#21262d", text: "#8b949e" },
};

export default async function ContentDetailPage({
  params,
}: PageProps<"/contents/[shortId]">) {
  const { shortId } = await params;
  const [content, variants] = await Promise.all([
    api.contents.get(shortId).catch(() => null),
    api.contents.variants(shortId).catch(() => []),
  ]);

  if (!content) notFound();

  const isVideo = content.content_type === "video";
  const isVr = content.content_type === "vr360";
  const statusStyle = STATUS_STYLE[content.status] ?? STATUS_STYLE.pending;

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div className="rounded-xl overflow-hidden" style={{ background: "#1c2128" }}>
        {isVr && variants.length > 0 ? (
          <Suspense fallback={<div className="aspect-video animate-pulse" style={{ background: "#21262d" }} />}>
            <VrViewer variants={variants} />
          </Suspense>
        ) : isVideo && variants.length > 0 ? (
          <Suspense fallback={<div className="aspect-video animate-pulse" style={{ background: "#21262d" }} />}>
            <HlsPlayer variants={variants} />
          </Suspense>
        ) : content.thumbnail_key ? (
          <div className="relative aspect-video">
            <Image
              src={thumbnailUrl(content.thumbnail_key)}
              alt={content.title}
              fill
              className="object-cover"
              unoptimized
            />
          </div>
        ) : (
          <div className="aspect-video flex flex-col items-center justify-center gap-3" style={{ color: "var(--gh-muted)" }}>
            <span className="text-6xl">{TYPE_ICONS[content.content_type] ?? "▪"}</span>
            <span className="text-sm">
              {content.status === "processing" ? "トランスコード処理中..." : "メディアなし"}
            </span>
          </div>
        )}
      </div>

      <div className="space-y-4">
        <h1 className="text-2xl font-bold" style={{ color: "var(--gh-text)" }}>{content.title}</h1>

        <div className="flex flex-wrap gap-2 text-sm">
          {content.disc_label && (
            <span style={{ color: "var(--gh-muted)" }}>ディスク: {content.disc_label}</span>
          )}
          {content.duration_seconds && (
            <span className="font-mono" style={{ color: "var(--gh-muted)" }}>{formatDuration(content.duration_seconds)}</span>
          )}
          {content.is_360 && (
            <span className="px-2 py-0.5 rounded text-xs font-medium" style={{ background: "#6e40c9", color: "#fff" }}>360°</span>
          )}
          <span className="px-2 py-0.5 rounded text-xs font-medium" style={{ background: statusStyle.bg, color: statusStyle.text }}>
            {content.status}
          </span>
        </div>

        {content.description && (
          <p className="whitespace-pre-wrap leading-relaxed" style={{ color: "var(--gh-muted)" }}>{content.description}</p>
        )}

        {content.tags.length > 0 && (
          <div className="flex flex-wrap gap-2">
            {content.tags.map((tag) => (
              <a
                key={tag}
                href={`/search?q=${encodeURIComponent(tag)}`}
                className="text-sm px-2 py-0.5 rounded transition-colors hover:opacity-80"
                style={{ background: "var(--tag-bg)", color: "var(--gh-accent)" }}
              >
                {tag}
              </a>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
