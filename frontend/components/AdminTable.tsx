'use client'

import { useState } from 'react'
import type { Content, ContentStatus } from '@/lib/types'
import EditModal from './EditModal'

interface Props {
  initialContents: Content[]
}

const STATUS_COLORS: Record<ContentStatus, { bg: string; text: string }> = {
  pending: { bg: '#d29922', text: '#fff' },
  processing: { bg: '#1f6feb', text: '#fff' },
  ready: { bg: '#238636', text: '#fff' },
  error: { bg: '#da3633', text: '#fff' },
}

const STATUS_LABEL: Record<ContentStatus, string> = {
  pending: 'Pending',
  processing: 'Processing',
  ready: 'Ready',
  error: 'Error',
}

const CONTENT_TYPE_LABEL: Record<string, string> = {
  video: 'Video',
  image_gallery: 'Gallery',
  vr360: 'VR360',
  document: 'Document',
}

export default function AdminTable({ initialContents }: Props) {
  const [contents, setContents] = useState<Content[]>(initialContents)
  const [editingContent, setEditingContent] = useState<Content | null>(null)

  function handleSave(updated: Content) {
    setContents((prev) =>
      prev.map((c) => (c.id === updated.id ? updated : c))
    )
    setEditingContent(null)
  }

  return (
    <>
      <div
        style={{
          overflowX: 'auto',
          border: '1px solid #30363d',
          borderRadius: 8,
        }}
      >
        <table
          style={{
            width: '100%',
            borderCollapse: 'collapse',
            fontSize: 14,
          }}
        >
          <thead>
            <tr
              style={{
                backgroundColor: '#161b22',
                borderBottom: '1px solid #30363d',
              }}
            >
              {['タイトル', '種別', 'ステータス', 'タグ', '更新日時', ''].map(
                (h) => (
                  <th
                    key={h}
                    style={{
                      padding: '10px 14px',
                      textAlign: 'left',
                      color: '#8b949e',
                      fontWeight: 500,
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {h}
                  </th>
                )
              )}
            </tr>
          </thead>
          <tbody>
            {contents.length === 0 && (
              <tr>
                <td
                  colSpan={6}
                  style={{
                    padding: '32px',
                    textAlign: 'center',
                    color: '#8b949e',
                  }}
                >
                  コンテンツがありません
                </td>
              </tr>
            )}
            {contents.map((content, i) => (
              <tr
                key={content.id}
                style={{
                  borderBottom:
                    i < contents.length - 1 ? '1px solid #30363d' : 'none',
                  backgroundColor: 'transparent',
                  transition: 'background 0.1s',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.backgroundColor = '#161b22'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.backgroundColor = 'transparent'
                }}
              >
                {/* Title */}
                <td
                  style={{
                    padding: '10px 14px',
                    color: '#e6edf3',
                    maxWidth: 280,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                  title={content.title}
                >
                  {content.title}
                </td>

                {/* Content type */}
                <td style={{ padding: '10px 14px', color: '#8b949e', whiteSpace: 'nowrap' }}>
                  {CONTENT_TYPE_LABEL[content.content_type] ?? content.content_type}
                </td>

                {/* Status badge */}
                <td style={{ padding: '10px 14px', whiteSpace: 'nowrap' }}>
                  <span
                    style={{
                      display: 'inline-block',
                      padding: '2px 8px',
                      borderRadius: 4,
                      fontSize: 12,
                      fontWeight: 600,
                      backgroundColor: STATUS_COLORS[content.status]?.bg ?? '#30363d',
                      color: STATUS_COLORS[content.status]?.text ?? '#e6edf3',
                    }}
                  >
                    {STATUS_LABEL[content.status] ?? content.status}
                  </span>
                </td>

                {/* Tags */}
                <td style={{ padding: '10px 14px' }}>
                  <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                    {content.tags.slice(0, 3).map((tag) => (
                      <span
                        key={tag}
                        style={{
                          fontSize: 11,
                          padding: '1px 6px',
                          borderRadius: 4,
                          backgroundColor: '#1f6feb22',
                          color: '#58a6ff',
                          border: '1px solid #1f6feb44',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {tag}
                      </span>
                    ))}
                    {content.tags.length > 3 && (
                      <span style={{ fontSize: 11, color: '#8b949e' }}>
                        +{content.tags.length - 3}
                      </span>
                    )}
                  </div>
                </td>

                {/* Updated at */}
                <td
                  style={{
                    padding: '10px 14px',
                    color: '#8b949e',
                    whiteSpace: 'nowrap',
                    fontSize: 12,
                  }}
                >
                  {new Date(content.updated_at).toLocaleDateString('ja-JP', {
                    year: 'numeric',
                    month: '2-digit',
                    day: '2-digit',
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </td>

                {/* Edit button */}
                <td style={{ padding: '10px 14px', whiteSpace: 'nowrap' }}>
                  <button
                    onClick={() => setEditingContent(content)}
                    style={{
                      padding: '4px 12px',
                      borderRadius: 6,
                      border: '1px solid #30363d',
                      backgroundColor: 'transparent',
                      color: '#8b949e',
                      cursor: 'pointer',
                      fontSize: 12,
                      transition: 'all 0.15s',
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.borderColor = '#58a6ff'
                      e.currentTarget.style.color = '#58a6ff'
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.borderColor = '#30363d'
                      e.currentTarget.style.color = '#8b949e'
                    }}
                  >
                    編集
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {editingContent && (
        <EditModal
          content={editingContent}
          onClose={() => setEditingContent(null)}
          onSave={handleSave}
        />
      )}
    </>
  )
}
