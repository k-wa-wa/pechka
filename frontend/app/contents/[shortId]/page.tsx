import { notFound } from 'next/navigation'
import { getContent, getVariants } from '@/lib/api'
import ContentPlayer from '@/components/ContentPlayer'
import type { Metadata } from 'next'
import type { MongoContent } from '@/lib/types'

export const dynamic = 'force-dynamic'

interface Props {
  params: Promise<{ shortId: string }>
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { shortId } = await params
  try {
    const content = await getContent(shortId)
    return { title: `${content.title} — pechka` }
  } catch {
    return { title: 'pechka' }
  }
}

export default async function ContentDetailPage({ params }: Props) {
  const { shortId } = await params

  let content: MongoContent
  let variants: Awaited<ReturnType<typeof getVariants>>
  try {
    ;[content, variants] = await Promise.all([
      getContent(shortId),
      getVariants(shortId),
    ])
  } catch {
    notFound()
  }

  const isVr = content.content_type === 'vr360'

  if (isVr) {
    return (
      <div>
        <ContentPlayer variants={variants} isVr={true} />
        <div
          style={{
            maxWidth: 900,
            margin: '0 auto',
            padding: '24px 24px',
          }}
        >
          <ContentInfo content={content} />
        </div>
      </div>
    )
  }

  return (
    <div
      style={{
        maxWidth: 960,
        margin: '0 auto',
        padding: '32px 24px',
      }}
    >
      <ContentPlayer variants={variants} isVr={false} />
      <div style={{ marginTop: 24 }}>
        <ContentInfo content={content} />
      </div>
    </div>
  )
}

function ContentInfo({ content }: { content: MongoContent }) {
  return (
    <>
      {/* Tags */}
      {content.tags.length > 0 && (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 16 }}>
          {content.tags.map((tag) => (
            <span
              key={tag}
              style={{
                fontSize: 12,
                padding: '3px 8px',
                borderRadius: 4,
                backgroundColor: '#1f6feb22',
                color: '#58a6ff',
                border: '1px solid #1f6feb44',
              }}
            >
              {tag}
            </span>
          ))}
        </div>
      )}

      {/* Title */}
      <h1
        style={{
          margin: '0 0 12px',
          fontSize: 'clamp(20px, 4vw, 28px)',
          fontWeight: 700,
          color: '#e6edf3',
          lineHeight: 1.3,
        }}
      >
        {content.title}
      </h1>

      {/* Meta */}
      <div
        style={{
          display: 'flex',
          gap: 16,
          marginBottom: 16,
          flexWrap: 'wrap',
          fontSize: 13,
          color: '#8b949e',
        }}
      >
        {content.duration_seconds != null && (
          <span>
            {Math.floor(content.duration_seconds / 60)}分{' '}
            {content.duration_seconds % 60}秒
          </span>
        )}
        {content.published_at && (
          <span>
            {new Date(content.published_at).toLocaleDateString('ja-JP')} 公開
          </span>
        )}
      </div>

      {/* Description */}
      {content.description && (
        <p
          style={{
            margin: 0,
            fontSize: 15,
            color: '#8b949e',
            lineHeight: 1.7,
            whiteSpace: 'pre-wrap',
          }}
        >
          {content.description}
        </p>
      )}
    </>
  )
}
