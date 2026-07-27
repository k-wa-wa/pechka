import { getContents } from '@/lib/api'
import type { ContentType } from '@/lib/types'
import HomeView from '@/components/HomeView'

// Force dynamic to always get fresh data
export const dynamic = 'force-dynamic'

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
    <HomeView
      carouselReady={carouselReady}
      gridItems={gridItems}
      currentType={params.type ?? ''}
    />
  )
}
