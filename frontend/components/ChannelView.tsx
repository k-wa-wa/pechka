'use client'

import Link from 'next/link'
import type { Channel, MongoContent } from '@/lib/types'
import ContentCard from '@/components/ContentCard'
import FilterBar from '@/components/FilterBar'
import { CONTENT_TYPES } from '@/components/HomeView'
import { useLanguage } from '@/lib/i18n/LanguageContext'

interface Props {
  channel: Channel
  contents: MongoContent[]
  currentType: string
}

export default function ChannelView({ channel, contents, currentType }: Props) {
  const { t } = useLanguage()

  return (
    <div
      style={{
        maxWidth: 1280,
        margin: '0 auto',
        padding: '32px 24px',
      }}
    >
      {/* Channel Header */}
      <div
        style={{
          backgroundColor: 'var(--card)',
          border: '1px solid var(--line)',
          borderRadius: 'var(--radius-lg)',
          padding: '24px',
          marginBottom: 32,
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            flexWrap: 'wrap',
            gap: 12,
            marginBottom: 8,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <h1
              style={{
                margin: 0,
                fontSize: 24,
                fontWeight: 700,
                color: 'var(--fg)',
              }}
            >
              {channel.name}
            </h1>
            <span
              style={{
                fontSize: 12,
                padding: '2px 8px',
                borderRadius: 'var(--radius-sm)',
                backgroundColor: 'var(--accent-subtle)',
                color: 'var(--accent)',
                border: '1px solid var(--accent-border)',
                fontFamily: 'monospace',
              }}
            >
              {channel.id}
            </span>
          </div>
          <Link
            href="/"
            style={{
              fontSize: 13,
              color: 'var(--muted)',
              textDecoration: 'none',
            }}
          >
            ← {t('nav.home')}
          </Link>
        </div>

        {channel.description && (
          <p
            style={{
              margin: '8px 0 0',
              fontSize: 14,
              color: 'var(--muted)',
              lineHeight: 1.6,
            }}
          >
            {channel.description}
          </p>
        )}
      </div>

      {/* Section header + filter */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          flexWrap: 'wrap',
          gap: 12,
          marginBottom: 24,
        }}
      >
        <h2
          style={{
            margin: 0,
            fontSize: 18,
            fontWeight: 700,
            color: 'var(--fg)',
          }}
        >
          {t('home.title')} ({contents.length})
        </h2>
        <FilterBar
          types={CONTENT_TYPES}
          currentType={currentType}
          basePath={`/channels/${channel.id}`}
        />
      </div>

      {contents.length === 0 ? (
        <div
          style={{
            textAlign: 'center',
            padding: '64px 0',
            color: 'var(--muted)',
          }}
        >
          {t('home.noContents')}
        </div>
      ) : (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))',
            gap: 16,
          }}
        >
          {contents.map((content) => (
            <ContentCard key={content.short_id} content={content} />
          ))}
        </div>
      )}
    </div>
  )
}
