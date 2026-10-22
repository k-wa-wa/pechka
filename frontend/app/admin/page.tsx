import { getAdminContents, getChannels } from '@/lib/api'
import AdminView from '@/components/AdminView'
import type { Metadata } from 'next'

export const dynamic = 'force-dynamic'

export const metadata: Metadata = {
  title: 'Admin — pechka',
}

const PAGE_SIZE = 20

export default async function AdminPage() {
  const [data, channels] = await Promise.all([
    getAdminContents({ limit: PAGE_SIZE, offset: 0 }).catch(() => ({
      contents: [],
      total: 0,
      limit: PAGE_SIZE,
      offset: 0,
    })),
    getChannels().catch(() => []),
  ])

  return (
    <AdminView
      contents={data.contents}
      total={data.total}
      limit={data.limit}
      offset={data.offset}
      channels={channels}
    />
  )
}
