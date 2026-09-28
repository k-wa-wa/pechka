'use client'

import { useEffect, useState } from 'react'
import type { Content, ContentStatus } from '@/lib/types'
import { archiveContent, getAdminContents, unarchiveContent } from '@/lib/api'
import EditModal from './EditModal'
import SubtitleEditorModal from './SubtitleEditorModal'
import UploadModal from './UploadModal'
import { useLanguage } from '@/lib/i18n/LanguageContext'

interface Props {
  initialContents: Content[]
  onUploaded?: () => void
}

const POLL_INTERVAL_MS = 5000
const IN_PROGRESS_STATUSES: ContentStatus[] = ['pending', 'processing']

const STATUS_COLORS: Record<ContentStatus, { bg: string; text: string; border: string }> = {
  pending: { bg: 'var(--blocked-subtle)', text: 'var(--blocked)', border: 'var(--blocked-border)' },
  processing: { bg: 'var(--accent-subtle)', text: 'var(--accent)', border: 'var(--accent-border)' },
  ready: { bg: 'var(--success-subtle)', text: 'var(--success)', border: 'var(--success-border)' },
  error: { bg: 'var(--warn-subtle)', text: 'var(--warn)', border: 'var(--warn-border)' },
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

export default function AdminTable({ initialContents, onUploaded }: Props) {
  const [contents, setContents] = useState<Content[]>(initialContents)
  const [editingContent, setEditingContent] = useState<Content | null>(null)
  const [subtitleContent, setSubtitleContent] = useState<Content | null>(null)
  const [archivingId, setArchivingId] = useState<string | null>(null)
  const [uploadOpen, setUploadOpen] = useState(false)
  const { t, language } = useLanguage()

  // Poll while any row is still converting so pending/processing rows flip to
  // ready (or error) without requiring a manual page reload.
  useEffect(() => {
    if (!contents.some((c) => IN_PROGRESS_STATUSES.includes(c.status))) return

    const timer = setTimeout(async () => {
      try {
        const res = await getAdminContents({ limit: 100, offset: 0 })
        const byId = new Map(res.contents.map((c) => [c.id, c]))
        setContents((prev) => prev.map((c) => byId.get(c.id) ?? c))
      } catch {
        // Ignore transient poll failures; the next tick will retry.
      }
    }, POLL_INTERVAL_MS)

    return () => clearTimeout(timer)
  }, [contents])

  function handleSave(updated: Content) {
    setContents((prev) =>
      prev.map((c) => (c.id === updated.id ? updated : c))
    )
    setEditingContent(null)
  }

  function handleUploaded(content: Content) {
    setContents((prev) => [content, ...prev])
    setUploadOpen(false)
    onUploaded?.()
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
          display: 'flex',
          justifyContent: 'flex-end',
          marginBottom: 12,
        }}
      >
        <button
          onClick={() => setUploadOpen(true)}
          className="admin-btn-primary"
          style={{
            padding: '6px 14px',
            borderRadius: 'var(--radius-md)',
            border: 'none',
            backgroundColor: 'var(--accent)',
            color: '#191816',
            cursor: 'pointer',
            fontSize: 13,
            fontWeight: 600,
            transition: 'opacity 0.15s, transform var(--duration-release) var(--ease-spring)',
          }}
        >
          {t('admin.table.btnUpload')}
        </button>
      </div>

      <div
        style={{
          overflowX: 'auto',
          border: '1px solid var(--line)',
          borderRadius: 'var(--radius-md)',
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
                backgroundColor: 'var(--card)',
                borderBottom: '1px solid var(--line)',
              }}
            >
              {tableHeaders.map((h, idx) => (
                <th
                  key={idx}
                  style={{
                    padding: '10px 14px',
                    textAlign: 'left',
                    color: 'var(--muted)',
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
                    color: 'var(--muted)',
                  }}
                >
                  {t('admin.table.noContents')}
                </td>
              </tr>
            )}
            {contents.map((content, i) => (
              <tr
                key={content.id}
                className="admin-table-row"
                style={{
                  borderBottom:
                    i < contents.length - 1 ? '1px solid var(--line)' : 'none',
                  backgroundColor: 'transparent',
                  transition: 'background 0.1s',
                }}
              >
                {/* Title */}
                <td
                  style={{
                    padding: '10px 14px',
                    color: 'var(--fg)',
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
                <td style={{ padding: '10px 14px', color: 'var(--muted)', whiteSpace: 'nowrap' }}>
                  {CONTENT_TYPE_LABEL[content.content_type] ?? content.content_type}
                </td>

                {/* Status badge */}
                <td style={{ padding: '10px 14px', whiteSpace: 'nowrap' }}>
                  <span
                    style={{
                      display: 'inline-block',
                      padding: '2px 8px',
                      borderRadius: 'var(--radius-sm)',
                      fontSize: 12,
                      fontWeight: 600,
                      backgroundColor: STATUS_COLORS[content.status]?.bg ?? 'var(--muted-subtle)',
                      color: STATUS_COLORS[content.status]?.text ?? 'var(--fg)',
                      border: `1px solid ${STATUS_COLORS[content.status]?.border ?? 'transparent'}`,
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
                        borderRadius: 'var(--radius-sm)',
                        fontSize: 12,
                        fontWeight: 600,
                        backgroundColor: 'var(--muted-subtle)',
                        color: 'var(--muted)',
                        border: '1px solid var(--muted-border)',
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
                          borderRadius: 'var(--radius-sm)',
                          backgroundColor: 'var(--accent-subtle)',
                          color: 'var(--accent)',
                          border: '1px solid var(--accent-border)',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {tag}
                      </span>
                    ))}
                    {content.tags.length > 3 && (
                      <span style={{ fontSize: 11, color: 'var(--muted)' }}>
                        +{content.tags.length - 3}
                      </span>
                    )}
                  </div>
                </td>

                {/* Updated at */}
                <td
                  style={{
                    padding: '10px 14px',
                    color: 'var(--muted)',
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

                {/* Action buttons */}
                <td style={{ padding: '10px 14px', whiteSpace: 'nowrap' }}>
                  <button
                    onClick={() => setEditingContent(content)}
                    className="admin-action-btn"
                    style={{
                      padding: '4px 12px',
                      borderRadius: 'var(--radius-md)',
                      border: '1px solid var(--line)',
                      backgroundColor: 'transparent',
                      color: 'var(--muted)',
                      cursor: 'pointer',
                      fontSize: 12,
                      transition: 'all 0.15s, transform var(--duration-release) var(--ease-spring)',
                    }}
                  >
                    {t('admin.table.btnEdit')}
                  </button>
                  <button
                    onClick={() => setSubtitleContent(content)}
                    className="admin-action-btn"
                    style={{
                      marginLeft: 6,
                      padding: '4px 12px',
                      borderRadius: 'var(--radius-md)',
                      border: '1px solid var(--line)',
                      backgroundColor: 'transparent',
                      color: 'var(--muted)',
                      cursor: 'pointer',
                      fontSize: 12,
                      transition: 'all 0.15s, transform var(--duration-release) var(--ease-spring)',
                    }}
                  >
                    {t('admin.table.btnSubtitles')}
                  </button>
                  <button
                    onClick={() => handleToggleArchive(content)}
                    disabled={archivingId === content.id}
                    className="admin-action-btn admin-archive-btn"
                    style={{
                      marginLeft: 6,
                      padding: '4px 12px',
                      borderRadius: 'var(--radius-md)',
                      border: '1px solid var(--line)',
                      backgroundColor: 'transparent',
                      color: archivingId === content.id ? 'var(--muted-border)' : 'var(--muted)',
                      cursor: archivingId === content.id ? 'not-allowed' : 'pointer',
                      fontSize: 12,
                      transition: 'all 0.15s, transform var(--duration-release) var(--ease-spring)',
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

      <style>{`
        @media (hover: hover) {
          .admin-table-row:hover {
            background-color: var(--card) !important;
          }
          .admin-action-btn:hover {
            border-color: var(--accent) !important;
            color: var(--accent) !important;
          }
          .admin-archive-btn:hover {
            border-color: var(--warn) !important;
            color: var(--warn) !important;
          }
          .admin-btn-primary:hover {
            opacity: 0.9;
          }
        }
        .admin-action-btn:active, .admin-btn-primary:active {
          transform: scale(var(--scale-button-active));
          transition: transform var(--duration-press) var(--ease-snappy);
        }
      `}</style>

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

      {uploadOpen && (
        <UploadModal onClose={() => setUploadOpen(false)} onUploaded={handleUploaded} />
      )}
    </>
  )
}
