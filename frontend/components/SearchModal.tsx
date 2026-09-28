'use client'

import { useState, useEffect, useCallback, useRef } from 'react'
import Link from 'next/link'
import type { SearchResult, ContentType } from '@/lib/types'
import { useLanguage } from '@/lib/i18n/LanguageContext'

const CONTENT_TYPE_LABEL: Record<ContentType, string> = {
  video: 'Video',
  image_gallery: 'Gallery',
  vr360: 'VR360',
  document: 'Document',
}

interface Props {
  isOpen: boolean
  onClose: () => void
}

export default function SearchModal({ isOpen, onClose }: Props) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResult[]>([])
  const [loading, setLoading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const { t } = useLanguage()

  useEffect(() => {
    if (isOpen) {
      setQuery('')
      setResults([])
      setTimeout(() => inputRef.current?.focus(), 50)
    }
  }, [isOpen])

  const doSearch = useCallback(async (q: string) => {
    if (!q.trim()) {
      setResults([])
      return
    }
    setLoading(true)
    try {
      const res = await fetch(
        `/api/v1/search?q=${encodeURIComponent(q)}&limit=10`
      )
      if (res.ok) {
        const data = (await res.json()) as SearchResult[]
        setResults(data)
      }
    } catch {
      setResults([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current)
    if (query.trim()) {
      setLoading(true)
    } else {
      setLoading(false)
      setResults([])
    }
    debounceRef.current = setTimeout(() => {
      doSearch(query)
    }, 300)
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
    }
  }, [query, doSearch])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        if (!isOpen) onClose() // toggle handled in header
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [isOpen, onClose])

  if (!isOpen) return null

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 1000,
        display: 'flex',
        alignItems: 'flex-start',
        justifyContent: 'center',
        paddingTop: '15vh',
        backgroundColor: 'rgba(0,0,0,0.6)',
      }}
      onClick={onClose}
    >
      <div
        style={{
          width: '100%',
          maxWidth: 600,
          margin: '0 16px',
          backgroundColor: 'var(--card)',
          border: '1px solid var(--line)',
          borderRadius: 'var(--radius-lg)',
          overflow: 'hidden',
          boxShadow: '0 24px 48px rgba(0,0,0,0.6)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Search input */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            padding: '12px 16px',
            borderBottom: '1px solid var(--line)',
            gap: 10,
          }}
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="var(--muted)"
            strokeWidth="2"
          >
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.35-4.35" />
          </svg>
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t('search.placeholder')}
            style={{
              flex: 1,
              background: 'transparent',
              border: 'none',
              outline: 'none',
              color: 'var(--fg)',
              fontSize: 16,
            }}
          />

          <kbd
            style={{
              color: 'var(--muted)',
              fontSize: 11,
              border: '1px solid var(--line)',
              borderRadius: 'var(--radius-sm)',
              padding: '2px 6px',
            }}
          >
            ESC
          </kbd>
        </div>

        {/* Results / Loading / No Results */}
        {loading && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              padding: '32px 24px',
              gap: 12,
              color: 'var(--muted)',
              fontSize: 14,
            }}
          >
            <style>{`
              @keyframes search-spin {
                to { transform: rotate(360deg); }
              }
            `}</style>
            <div
              style={{
                width: 20,
                height: 20,
                border: '2px solid var(--line)',
                borderTopColor: 'var(--accent)',
                borderRadius: '50%',
                animation: 'search-spin 0.8s linear infinite',
              }}
            />
            <span>{t('search.searching')}</span>
          </div>
        )}

        {!loading && results.length > 0 && (
          <ul style={{ listStyle: 'none', margin: 0, padding: '8px 0', maxHeight: 400, overflowY: 'auto' }}>
            {results.map((r) => (
              <li key={r.short_id}>
                <Link
                  href={`/contents/${r.short_id}`}
                  onClick={onClose}
                  className="search-result-item"
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 12,
                    padding: '10px 16px',
                    transition: 'background 0.15s',
                  }}
                >
                  <span
                    style={{
                      fontSize: 11,
                      padding: '2px 6px',
                      borderRadius: 'var(--radius-sm)',
                      backgroundColor: 'var(--accent-emphasis)',
                      color: 'var(--accent)',
                      border: '1px solid var(--accent-border)',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {CONTENT_TYPE_LABEL[r.content_type] ?? r.content_type}
                  </span>
                  <span style={{ flex: 1, color: 'var(--fg)', fontSize: 14 }}>
                    {r.title}
                  </span>
                  {r.tags.slice(0, 2).map((tag) => (
                    <span
                      key={tag}
                      style={{
                        fontSize: 11,
                        padding: '1px 6px',
                        borderRadius: 'var(--radius-sm)',
                        backgroundColor: 'var(--accent-subtle)',
                        color: 'var(--accent)',
                        border: '1px solid var(--accent-border)',
                      }}
                    >
                      {tag}
                    </span>
                  ))}
                </Link>
              </li>
            ))}
          </ul>
        )}

        {query && !loading && results.length === 0 && (
          <div
            style={{
              padding: '24px',
              textAlign: 'center',
              color: 'var(--muted)',
              fontSize: 14,
            }}
          >
            {t('search.noResultsPrefix')}{query}{t('search.noResultsSuffix')}
          </div>
        )}
      </div>

      <style>{`
        @media (hover: hover) {
          .search-result-item:hover {
            background-color: var(--bg) !important;
          }
        }
      `}</style>
    </div>
  )
}
