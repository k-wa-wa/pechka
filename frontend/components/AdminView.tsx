'use client'

import { useState } from 'react'
import type { Content } from '@/lib/types'
import { getAdminContents } from '@/lib/api'
import AdminTable from '@/components/AdminTable'
import { useLanguage } from '@/lib/i18n/LanguageContext'

interface Props {
  contents: Content[]
  total: number
  limit: number
  offset: number
}

export default function AdminView({ contents, total, limit, offset }: Props) {
  const { t } = useLanguage()
  const [items, setItems] = useState(contents)
  const [currentOffset, setCurrentOffset] = useState(offset)
  const [currentTotal, setCurrentTotal] = useState(total)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(false)

  const totalPages = Math.max(1, Math.ceil(currentTotal / limit))
  const currentPage = Math.floor(currentOffset / limit) + 1

  async function goToPage(page: number) {
    const clamped = Math.min(Math.max(page, 1), totalPages)
    const newOffset = (clamped - 1) * limit
    if (newOffset === currentOffset) return

    setLoading(true)
    setError(false)
    try {
      const res = await getAdminContents({ limit, offset: newOffset })
      setItems(res.contents)
      setCurrentOffset(res.offset)
      setCurrentTotal(res.total)
    } catch {
      setError(true)
    } finally {
      setLoading(false)
    }
  }

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
          {currentTotal} {t('admin.itemsCount')}
        </span>
      </div>

      <AdminTable
        key={currentOffset}
        initialContents={items}
        onUploaded={() => setCurrentTotal((t) => t + 1)}
      />

      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 12,
          marginTop: 20,
        }}
      >
        <button
          onClick={() => goToPage(currentPage - 1)}
          disabled={loading || currentPage <= 1}
          style={{
            padding: '6px 14px',
            borderRadius: 6,
            border: '1px solid #30363d',
            backgroundColor: 'transparent',
            color: currentPage <= 1 ? '#8b949e88' : '#e6edf3',
            cursor: loading || currentPage <= 1 ? 'not-allowed' : 'pointer',
            fontSize: 13,
          }}
        >
          {t('admin.pagination.prev')}
        </button>

        <span style={{ fontSize: 13, color: '#8b949e' }}>
          {t('admin.pagination.pageLabel')} {currentPage} / {totalPages}
        </span>

        <button
          onClick={() => goToPage(currentPage + 1)}
          disabled={loading || currentPage >= totalPages}
          style={{
            padding: '6px 14px',
            borderRadius: 6,
            border: '1px solid #30363d',
            backgroundColor: 'transparent',
            color: currentPage >= totalPages ? '#8b949e88' : '#e6edf3',
            cursor: loading || currentPage >= totalPages ? 'not-allowed' : 'pointer',
            fontSize: 13,
          }}
        >
          {t('admin.pagination.next')}
        </button>
      </div>

      {error && (
        <div
          style={{
            textAlign: 'center',
            marginTop: 12,
            fontSize: 13,
            color: '#da3633',
          }}
        >
          {t('admin.pagination.loadError')}
        </div>
      )}
    </div>
  )
}
