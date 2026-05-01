'use client'

import { useState, useEffect, useCallback, useRef } from 'react'
import Link from 'next/link'
import type { SearchResult, ContentType } from '@/lib/types'

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
          backgroundColor: '#161b22',
          border: '1px solid #30363d',
          borderRadius: 12,
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
            borderBottom: '1px solid #30363d',
            gap: 10,
          }}
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="#8b949e"
            strokeWidth="2"
          >
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.35-4.35" />
          </svg>
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="コンテンツを検索..."
            style={{
              flex: 1,
              background: 'transparent',
              border: 'none',
              outline: 'none',
              color: '#e6edf3',
              fontSize: 16,
            }}
          />
          {loading && (
            <span style={{ color: '#8b949e', fontSize: 12 }}>...</span>
          )}
          <kbd
            style={{
              color: '#8b949e',
              fontSize: 11,
              border: '1px solid #30363d',
              borderRadius: 4,
              padding: '2px 6px',
            }}
          >
            ESC
          </kbd>
        </div>

        {/* Results */}
        {results.length > 0 && (
          <ul style={{ listStyle: 'none', margin: 0, padding: '8px 0', maxHeight: 400, overflowY: 'auto' }}>
            {results.map((r) => (
              <li key={r.short_id}>
                <Link
                  href={`/contents/${r.short_id}`}
                  onClick={onClose}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 12,
                    padding: '10px 16px',
                    transition: 'background 0.15s',
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.backgroundColor = '#0d1117'
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.backgroundColor = 'transparent'
                  }}
                >
                  <span
                    style={{
                      fontSize: 11,
                      padding: '2px 6px',
                      borderRadius: 4,
                      backgroundColor: '#1f6feb33',
                      color: '#58a6ff',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {CONTENT_TYPE_LABEL[r.content_type] ?? r.content_type}
                  </span>
                  <span style={{ flex: 1, color: '#e6edf3', fontSize: 14 }}>
                    {r.title}
                  </span>
                  {r.tags.slice(0, 2).map((tag) => (
                    <span
                      key={tag}
                      style={{
                        fontSize: 11,
                        padding: '1px 6px',
                        borderRadius: 4,
                        backgroundColor: '#1f6feb22',
                        color: '#58a6ff',
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
              color: '#8b949e',
              fontSize: 14,
            }}
          >
            「{query}」に一致するコンテンツが見つかりません
          </div>
        )}
      </div>
    </div>
  )
}
