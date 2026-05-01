import Link from 'next/link'
import type { MongoContent, ContentType } from '@/lib/types'

const CONTENT_TYPE_LABEL: Record<ContentType, string> = {
  video: 'Video',
  image_gallery: 'Gallery',
  vr360: 'VR360',
  document: 'Document',
}

const CONTENT_TYPE_ICON: Record<ContentType, React.ReactNode> = {
  video: (
    <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#8b949e" strokeWidth="1.5">
      <polygon points="5 3 19 12 5 21 5 3" />
    </svg>
  ),
  image_gallery: (
    <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#8b949e" strokeWidth="1.5">
      <rect x="3" y="3" width="18" height="18" rx="2" />
      <circle cx="8.5" cy="8.5" r="1.5" />
      <polyline points="21 15 16 10 5 21" />
    </svg>
  ),
  vr360: (
    <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#8b949e" strokeWidth="1.5">
      <circle cx="12" cy="12" r="10" />
      <ellipse cx="12" cy="12" rx="10" ry="4" />
      <line x1="12" y1="2" x2="12" y2="22" />
    </svg>
  ),
  document: (
    <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#8b949e" strokeWidth="1.5">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <polyline points="14 2 14 8 20 8" />
    </svg>
  ),
}

function formatDuration(seconds: number): string {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  return `${m}:${String(s).padStart(2, '0')}`
}

interface Props {
  content: MongoContent
}

export default function ContentCard({ content }: Props) {
  const thumbnailUrl = content.thumbnail_key
    ? `/thumbnails/${content.thumbnail_key}`
    : null

  return (
    <Link
      href={`/contents/${content.short_id}`}
      style={{
        display: 'block',
        backgroundColor: '#161b22',
        border: '1px solid #30363d',
        borderRadius: 10,
        overflow: 'hidden',
        transition: 'border-color 0.15s, transform 0.15s',
        cursor: 'pointer',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.borderColor = '#58a6ff'
        e.currentTarget.style.transform = 'translateY(-2px)'
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.borderColor = '#30363d'
        e.currentTarget.style.transform = 'translateY(0)'
      }}
    >
      {/* Thumbnail */}
      <div
        style={{
          width: '100%',
          aspectRatio: '16/9',
          backgroundColor: '#0d1117',
          position: 'relative',
          overflow: 'hidden',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        {thumbnailUrl ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={thumbnailUrl}
            alt={content.title}
            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
          />
        ) : (
          CONTENT_TYPE_ICON[content.content_type]
        )}

        {/* Duration badge */}
        {content.duration_seconds != null && (
          <span
            style={{
              position: 'absolute',
              bottom: 8,
              right: 8,
              backgroundColor: 'rgba(0,0,0,0.8)',
              color: '#e6edf3',
              fontSize: 12,
              padding: '2px 6px',
              borderRadius: 4,
              fontVariantNumeric: 'tabular-nums',
            }}
          >
            {formatDuration(content.duration_seconds)}
          </span>
        )}

        {/* Content type badge */}
        <span
          style={{
            position: 'absolute',
            top: 8,
            left: 8,
            backgroundColor: 'rgba(31,111,235,0.85)',
            color: '#58a6ff',
            fontSize: 11,
            padding: '2px 6px',
            borderRadius: 4,
          }}
        >
          {CONTENT_TYPE_LABEL[content.content_type]}
        </span>
      </div>

      {/* Info */}
      <div style={{ padding: '12px' }}>
        <h3
          style={{
            margin: 0,
            fontSize: 14,
            fontWeight: 600,
            color: '#e6edf3',
            lineHeight: 1.4,
            overflow: 'hidden',
            display: '-webkit-box',
            WebkitLineClamp: 2,
            WebkitBoxOrient: 'vertical',
          }}
        >
          {content.title}
        </h3>

        {content.tags.length > 0 && (
          <div
            style={{
              display: 'flex',
              flexWrap: 'wrap',
              gap: 4,
              marginTop: 8,
            }}
          >
            {content.tags.slice(0, 3).map((tag) => (
              <span
                key={tag}
                style={{
                  fontSize: 11,
                  padding: '2px 6px',
                  borderRadius: 4,
                  backgroundColor: '#1f6feb22',
                  color: '#58a6ff',
                  border: '1px solid #1f6feb44',
                }}
              >
                {tag}
              </span>
            ))}
            {content.tags.length > 3 && (
              <span style={{ fontSize: 11, color: '#8b949e', padding: '2px 0' }}>
                +{content.tags.length - 3}
              </span>
            )}
          </div>
        )}
      </div>
    </Link>
  )
}
