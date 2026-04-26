import Link from "next/link";
import Image from "next/image";
import { type MongoContent } from "@/lib/api";
import { thumbnailUrl, formatDuration } from "@/lib/utils";

interface Props {
  content: MongoContent;
}

export function ContentCard({ content }: Props) {
  return (
    <Link href={`/contents/${content.short_id}`} className="group block rounded-lg overflow-hidden border border-gray-200 hover:border-blue-400 transition-colors">
      <div className="relative aspect-video bg-gray-100">
        <Image
          src={thumbnailUrl(content.thumbnail_key)}
          alt={content.title}
          fill
          className="object-cover"
          unoptimized
        />
        {content.duration_seconds && (
          <span className="absolute bottom-1 right-1 bg-black/70 text-white text-xs px-1 rounded">
            {formatDuration(content.duration_seconds)}
          </span>
        )}
        {content.is_360 && (
          <span className="absolute top-1 left-1 bg-purple-600 text-white text-xs px-1 rounded">360°</span>
        )}
      </div>
      <div className="p-3">
        <h3 className="font-medium text-gray-900 line-clamp-2 group-hover:text-blue-600">{content.title}</h3>
        {content.tags.length > 0 && (
          <div className="flex flex-wrap gap-1 mt-1">
            {content.tags.slice(0, 3).map((tag) => (
              <span key={tag} className="text-xs bg-gray-100 text-gray-600 px-1.5 py-0.5 rounded">
                {tag}
              </span>
            ))}
          </div>
        )}
      </div>
    </Link>
  );
}
