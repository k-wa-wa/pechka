'use client'

import type { MongoContent, ContentType } from '@/lib/types'
import ContentCard from '@/components/ContentCard'
import Carousel from '@/components/Carousel'
import FilterBar, { FilterOption } from '@/components/FilterBar'
import { useLanguage } from '@/lib/i18n/LanguageContext'

const CONTENT_TYPES: FilterOption[] = [
  { value: '', labelKey: 'filter.all' },
  { value: 'video', label: 'Video' },
  { value: 'image_gallery', label: 'Gallery' },
  { value: 'vr360', label: 'VR360' },
  { value: 'document', label: 'Document' },
]

interface Props {
  carouselReady: MongoContent[]
  gridItems: MongoContent[]
  currentType: string
}

export default function HomeView({ carouselReady, gridItems, currentType }: Props) {
  const { t } = useLanguage()

  return (
    <div>
      {/* Carousel */}
      {carouselReady.length > 0 && <Carousel items={carouselReady} />}

      {/* Main content area */}
      <div
        style={{
          maxWidth: 1280,
          margin: '0 auto',
          padding: '32px 24px',
        }}
      >
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
          <h1
            style={{
              margin: 0,
              fontSize: 20,
              fontWeight: 700,
              color: '#e6edf3',
            }}
          >
            {t('home.title')}
          </h1>
          <FilterBar types={CONTENT_TYPES} currentType={currentType} />
        </div>

        {gridItems.length === 0 ? (
          <div
            style={{
              textAlign: 'center',
              padding: '64px 0',
              color: '#8b949e',
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
            {gridItems.map((content) => (
              <ContentCard key={content.short_id} content={content} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
