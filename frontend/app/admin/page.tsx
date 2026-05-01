import { getAdminContents } from '@/lib/api'
import AdminTable from '@/components/AdminTable'
import type { Metadata } from 'next'

export const dynamic = 'force-dynamic'

export const metadata: Metadata = {
  title: 'Admin — pechka',
}

export default async function AdminPage() {
  const contents = await getAdminContents({ limit: 200 }).catch(() => [])

  return (
    <div
      style={{
        maxWidth: 1280,
        margin: '0 auto',
        padding: '32px 24px',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 24,
        }}
      >
        <h1
          style={{
            margin: 0,
            fontSize: 22,
            fontWeight: 700,
            color: '#e6edf3',
          }}
        >
          管理画面
        </h1>
        <span style={{ fontSize: 14, color: '#8b949e' }}>
          {contents.length} 件
        </span>
      </div>

      <AdminTable initialContents={contents} />
    </div>
  )
}
