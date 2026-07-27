import { getAdminContents } from '@/lib/api'
import AdminView from '@/components/AdminView'
import type { Metadata } from 'next'

export const dynamic = 'force-dynamic'

export const metadata: Metadata = {
  title: 'Admin — pechka',
}

export default async function AdminPage() {
  const contents = await getAdminContents({ limit: 200 }).catch(() => [])

  return <AdminView contents={contents} />
}
