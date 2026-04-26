import { notFound } from "next/navigation";
import { Suspense } from "react";
import { api } from "@/lib/api";
import { HlsPlayer } from "@/components/HlsPlayer";
import { VrViewer } from "@/components/VrViewer";
import { thumbnailUrl, formatDuration } from "@/lib/utils";
import Image from "next/image";

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

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div>
        {isVr && variants.length > 0 ? (
          <Suspense fallback={<div className="aspect-video bg-gray-200 rounded-lg animate-pulse" />}>
            <VrViewer variants={variants} />
          </Suspense>
        ) : isVideo && variants.length > 0 ? (
          <Suspense fallback={<div className="aspect-video bg-gray-200 rounded-lg animate-pulse" />}>
            <HlsPlayer variants={variants} />
          </Suspense>
        ) : content.thumbnail_key ? (
          <div className="relative aspect-video bg-gray-100 rounded-lg overflow-hidden">
            <Image
              src={thumbnailUrl(content.thumbnail_key)}
              alt={content.title}
              fill
              className="object-cover"
              unoptimized
            />
          </div>
        ) : null}
      </div>

      <div className="space-y-3">
        <h1 className="text-2xl font-bold text-gray-900">{content.title}</h1>

        <div className="flex flex-wrap gap-2 text-sm text-gray-500">
          {content.disc_label && <span>ディスク: {content.disc_label}</span>}
          {content.duration_seconds && <span>{formatDuration(content.duration_seconds)}</span>}
          {content.is_360 && <span className="bg-purple-100 text-purple-700 px-2 py-0.5 rounded">360°</span>}
          <span className={`px-2 py-0.5 rounded ${content.status === "ready" ? "bg-green-100 text-green-700" : "bg-yellow-100 text-yellow-700"}`}>
            {content.status}
          </span>
        </div>

        {content.description && (
          <p className="text-gray-700 whitespace-pre-wrap">{content.description}</p>
        )}

        {content.tags.length > 0 && (
          <div className="flex flex-wrap gap-2">
            {content.tags.map((tag) => (
              <a
                key={tag}
                href={`/search?q=${encodeURIComponent(tag)}`}
                className="text-sm bg-gray-100 text-gray-700 px-2 py-0.5 rounded hover:bg-blue-100 hover:text-blue-700 transition-colors"
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
