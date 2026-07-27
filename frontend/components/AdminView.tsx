'use client'

import type { Content } from '@/lib/types'
import AdminTable from '@/components/AdminTable'
import { useLanguage } from '@/lib/i18n/LanguageContext'

interface Props {
  contents: Content[]
}

export default function AdminView({ contents }: Props) {
  const { t } = useLanguage()

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
          {t('admin.title')}
        </h1>
        <span style={{ fontSize: 14, color: '#8b949e' }}>
          {contents.length} {t('admin.itemsCount')}
        </span>
      </div>

      <AdminTable initialContents={contents} />
    </div>
  )
}
