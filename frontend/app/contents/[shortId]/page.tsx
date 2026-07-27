import { notFound } from 'next/navigation'
import { getContent, getVariants } from '@/lib/api'
import ContentDetailView from '@/components/ContentDetailView'
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

  return <ContentDetailView content={content} variants={variants} />
}
