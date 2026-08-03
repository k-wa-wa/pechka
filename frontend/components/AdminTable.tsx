'use client'

import { useState } from 'react'
import type { Content, ContentStatus } from '@/lib/types'
import { archiveContent, unarchiveContent } from '@/lib/api'
import EditModal from './EditModal'
import SubtitleEditorModal from './SubtitleEditorModal'
import { useLanguage } from '@/lib/i18n/LanguageContext'

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
  const [subtitleContent, setSubtitleContent] = useState<Content | null>(null)
  const [archivingId, setArchivingId] = useState<string | null>(null)
  const { t, language } = useLanguage()

  function handleSave(updated: Content) {
    setContents((prev) =>
      prev.map((c) => (c.id === updated.id ? updated : c))
    )
    setEditingContent(null)
  }

  async function handleToggleArchive(content: Content) {
    const archiving = !content.archived_at
    const confirmMessage = archiving
      ? t('admin.table.confirmArchive')
      : t('admin.table.confirmUnarchive')
    if (!window.confirm(confirmMessage)) return

    setArchivingId(content.id)
    try {
      const updated = archiving
        ? await archiveContent(content.id)
        : await unarchiveContent(content.id)
      setContents((prev) => prev.map((c) => (c.id === updated.id ? updated : c)))
    } catch (e) {
      window.alert(e instanceof Error ? e.message : t('admin.table.archiveError'))
    } finally {
      setArchivingId(null)
    }
  }

  const tableHeaders = [
    t('admin.table.colTitle'),
    t('admin.table.colType'),
    t('admin.table.colStatus'),
    t('admin.table.colTags'),
    t('admin.table.colUpdatedAt'),
    '',
  ]

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
              {tableHeaders.map((h, idx) => (
                <th
                  key={idx}
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
              ))}
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
                  {t('admin.table.noContents')}
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
                  {content.archived_at && (
                    <span
                      style={{
                        display: 'inline-block',
                        marginLeft: 6,
                        padding: '2px 8px',
                        borderRadius: 4,
                        fontSize: 12,
                        fontWeight: 600,
                        backgroundColor: '#30363d',
                        color: '#8b949e',
                      }}
                    >
                      {t('admin.table.badgeArchived')}
                    </span>
                  )}
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
                  {new Date(content.updated_at).toLocaleDateString(
                    language === 'ja' ? 'ja-JP' : 'en-US',
                    {
                      year: 'numeric',
                      month: '2-digit',
                      day: '2-digit',
                      hour: '2-digit',
                      minute: '2-digit',
                      timeZone: 'Asia/Tokyo',
                    }
                  )}
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
                    {t('admin.table.btnEdit')}
                  </button>
                  <button
                    onClick={() => setSubtitleContent(content)}
                    style={{
                      marginLeft: 6,
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
                    {t('admin.table.btnSubtitles')}
                  </button>
                  <button
                    onClick={() => handleToggleArchive(content)}
                    disabled={archivingId === content.id}
                    style={{
                      marginLeft: 6,
                      padding: '4px 12px',
                      borderRadius: 6,
                      border: '1px solid #30363d',
                      backgroundColor: 'transparent',
                      color: archivingId === content.id ? '#8b949e88' : '#8b949e',
                      cursor: archivingId === content.id ? 'not-allowed' : 'pointer',
                      fontSize: 12,
                      transition: 'all 0.15s',
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.borderColor = '#da3633'
                      e.currentTarget.style.color = '#da3633'
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.borderColor = '#30363d'
                      e.currentTarget.style.color =
                        archivingId === content.id ? '#8b949e88' : '#8b949e'
                    }}
                  >
                    {content.archived_at
                      ? t('admin.table.btnUnarchive')
                      : t('admin.table.btnArchive')}
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

      {subtitleContent && (
        <SubtitleEditorModal
          content={subtitleContent}
          onClose={() => setSubtitleContent(null)}
        />
      )}
    </>
  )
}
