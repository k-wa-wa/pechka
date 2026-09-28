'use client'

import type { MongoContent, MongoVariant } from '@/lib/types'
import ContentPlayer from '@/components/ContentPlayer'
import { useLanguage } from '@/lib/i18n/LanguageContext'

interface Props {
  content: MongoContent
  variants: MongoVariant[]
}

export default function ContentDetailView({ content, variants }: Props) {
  const { t, language } = useLanguage()
  const isVr = content.content_type === 'vr360'

  return (
    <div
      style={{
        maxWidth: isVr ? 1280 : 960,
        margin: '0 auto',
        padding: isVr ? '0 0 32px' : '32px 24px',
      }}
    >
      <ContentPlayer
        variants={variants}
        isVr={isVr}
        shortId={content.short_id}
        hasSubtitles={content.has_subtitles}
      />
      <div
        style={{
          maxWidth: isVr ? 900 : undefined,
          margin: isVr ? '24px auto 0' : '24px 0 0',
          padding: isVr ? '0 24px' : undefined,
        }}
      >
        {/* Tags */}
        {content.tags.length > 0 && (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 16 }}>
            {content.tags.map((tag) => (
              <span
                key={tag}
                style={{
                  fontSize: 12,
                  padding: '3px 8px',
                  borderRadius: 'var(--radius-sm)',
                  backgroundColor: 'var(--accent-subtle)',
                  color: 'var(--accent)',
                  border: '1px solid var(--accent-border)',
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
            color: 'var(--fg)',
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
            color: 'var(--muted)',
          }}
        >
          {content.duration_seconds != null && (
            <span>
              {Math.floor(content.duration_seconds / 60)}{t('content.minutes')}
              {content.duration_seconds % 60}{t('content.seconds')}
            </span>
          )}
          {content.published_at && (
            <span>
              {new Date(content.published_at).toLocaleDateString(
                language === 'ja' ? 'ja-JP' : 'en-US',
                { timeZone: 'Asia/Tokyo' }
              )}{' '}
              {t('content.published')}
            </span>
          )}
        </div>

        {/* Description */}
        {content.description && (
          <p
            style={{
              margin: 0,
              fontSize: 15,
              color: 'var(--muted)',
              lineHeight: 1.7,
              whiteSpace: 'pre-wrap',
            }}
          >
            {content.description}
          </p>
        )}
      </div>
    </div>
  )
}
