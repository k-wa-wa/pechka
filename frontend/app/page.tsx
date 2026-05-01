import { getContents } from '@/lib/api'
import type { ContentType } from '@/lib/types'
import ContentCard from '@/components/ContentCard'
import Carousel from '@/components/Carousel'
import FilterBar from '@/components/FilterBar'

// Force dynamic to always get fresh data
export const dynamic = 'force-dynamic'

const CONTENT_TYPES: { value: ContentType | ''; label: string }[] = [
  { value: '', label: 'すべて' },
  { value: 'video', label: 'Video' },
  { value: 'image_gallery', label: 'Gallery' },
  { value: 'vr360', label: 'VR360' },
  { value: 'document', label: 'Document' },
]

interface Props {
  searchParams: Promise<{ type?: string }>
}

export default async function HomePage({ searchParams }: Props) {
  const params = await searchParams
  const contentType = (params.type as ContentType) || undefined

  const [carouselItems, allContents] = await Promise.all([
    getContents({ limit: 8 }).catch(() => []),
    getContents({ limit: 100, content_type: contentType }).catch(() => []),
  ])

  // Carousel: most recent ready items
  const carouselReady = carouselItems
    .filter((c) => c.status === 'ready')
    .slice(0, 6)

  // Grid: ready items only
  const gridItems = allContents.filter((c) => c.status === 'ready')

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
            コンテンツ一覧
          </h1>
          <FilterBar types={CONTENT_TYPES} currentType={params.type ?? ''} />
        </div>

        {gridItems.length === 0 ? (
          <div
            style={{
              textAlign: 'center',
              padding: '64px 0',
              color: '#8b949e',
            }}
          >
            コンテンツがありません
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
