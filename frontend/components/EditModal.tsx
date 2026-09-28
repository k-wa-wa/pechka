'use client'

import { useState } from 'react'
import type { Content, ContentStatus, UpdateContentRequest } from '@/lib/types'
import { useLanguage } from '@/lib/i18n/LanguageContext'

interface Props {
  content: Content
  onClose: () => void
  onSave: (updated: Content) => void
}

const STATUS_OPTIONS: ContentStatus[] = ['pending', 'processing', 'ready', 'error']

export default function EditModal({ content, onClose, onSave }: Props) {
  const [title, setTitle] = useState(content.title)
  const [description, setDescription] = useState(content.description)
  const [tags, setTags] = useState(content.tags.join(', '))
  const [status, setStatus] = useState<ContentStatus>(content.status)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const { t } = useLanguage()

  async function handleSave() {
    setSaving(true)
    setError(null)
    try {
      const body: UpdateContentRequest = {
        title: title || null,
        description: description || null,
        tags: tags
          .split(',')
          .map((t) => t.trim())
          .filter(Boolean),
        status,
      }
      const res = await fetch(`/api/v1/admin/contents/${content.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
      const updated = (await res.json()) as Content
      onSave(updated)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 200,
        backgroundColor: 'rgba(0,0,0,0.7)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 16,
      }}
      onClick={onClose}
    >
      <div
        style={{
          width: '100%',
          maxWidth: 560,
          backgroundColor: 'var(--card)',
          border: '1px solid var(--line)',
          borderRadius: 'var(--radius-lg)',
          overflow: 'hidden',
          boxShadow: '0 24px 48px rgba(0,0,0,0.5)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '16px 20px',
            borderBottom: '1px solid var(--line)',
          }}
        >
          <h2 style={{ margin: 0, fontSize: 16, color: 'var(--fg)' }}>
            {t('editModal.title')}
          </h2>
          <button
            onClick={onClose}
            className="edit-modal-close"
            style={{
              background: 'none',
              border: 'none',
              color: 'var(--muted)',
              cursor: 'pointer',
              padding: 4,
              borderRadius: 'var(--radius-sm)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              transition: 'color 0.15s, transform var(--duration-release) var(--ease-spring)',
            }}
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Form */}
        <div style={{ padding: '20px', display: 'flex', flexDirection: 'column', gap: 16 }}>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <span style={{ fontSize: 13, color: 'var(--muted)' }}>{t('editModal.fieldTitle')}</span>
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="edit-form-input"
              style={{
                backgroundColor: 'var(--bg)',
                border: '1px solid var(--line)',
                borderRadius: 'var(--radius-md)',
                color: 'var(--fg)',
                padding: '8px 12px',
                fontSize: 14,
                outline: 'none',
              }}
            />
          </label>

          <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <span style={{ fontSize: 13, color: 'var(--muted)' }}>{t('editModal.fieldDescription')}</span>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={4}
              className="edit-form-input"
              style={{
                backgroundColor: 'var(--bg)',
                border: '1px solid var(--line)',
                borderRadius: 'var(--radius-md)',
                color: 'var(--fg)',
                padding: '8px 12px',
                fontSize: 14,
                outline: 'none',
                resize: 'vertical',
                fontFamily: 'inherit',
              }}
            />
          </label>

          <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <span style={{ fontSize: 13, color: 'var(--muted)' }}>
              {t('editModal.fieldTags')}
            </span>
            <input
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              placeholder="tag1, tag2, tag3"
              className="edit-form-input"
              style={{
                backgroundColor: 'var(--bg)',
                border: '1px solid var(--line)',
                borderRadius: 'var(--radius-md)',
                color: 'var(--fg)',
                padding: '8px 12px',
                fontSize: 14,
                outline: 'none',
              }}
            />
          </label>

          <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <span style={{ fontSize: 13, color: 'var(--muted)' }}>{t('editModal.fieldStatus')}</span>
            <select
              value={status}
              onChange={(e) => setStatus(e.target.value as ContentStatus)}
              className="edit-form-input"
              style={{
                backgroundColor: 'var(--bg)',
                border: '1px solid var(--line)',
                borderRadius: 'var(--radius-md)',
                color: 'var(--fg)',
                padding: '8px 12px',
                fontSize: 14,
                outline: 'none',
                cursor: 'pointer',
              }}
            >
              {STATUS_OPTIONS.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </select>
          </label>

          {error && (
            <div
              style={{
                backgroundColor: 'var(--warn-subtle)',
                border: '1px solid var(--warn-border)',
                borderRadius: 'var(--radius-md)',
                padding: '8px 12px',
                color: 'var(--warn)',
                fontSize: 13,
              }}
            >
              {error}
            </div>
          )}
        </div>

        {/* Footer */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            gap: 8,
            padding: '12px 20px',
            borderTop: '1px solid var(--line)',
          }}
        >
          <button
            onClick={onClose}
            disabled={saving}
            className="edit-cancel-btn"
            style={{
              padding: '8px 16px',
              borderRadius: 'var(--radius-md)',
              border: '1px solid var(--line)',
              backgroundColor: 'transparent',
              color: 'var(--muted)',
              cursor: 'pointer',
              fontSize: 14,
              transition: 'border-color 0.15s, color 0.15s, transform var(--duration-release) var(--ease-spring)',
            }}
          >
            {t('editModal.btnCancel')}
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="edit-save-btn"
            style={{
              padding: '8px 16px',
              borderRadius: 'var(--radius-md)',
              border: 'none',
              backgroundColor: saving ? 'var(--accent-emphasis)' : 'var(--accent)',
              color: saving ? 'var(--muted)' : '#191816',
              cursor: saving ? 'not-allowed' : 'pointer',
              fontSize: 14,
              fontWeight: 600,
              transition: 'opacity 0.15s, transform var(--duration-release) var(--ease-spring)',
            }}
          >
            {saving ? t('editModal.saving') : t('editModal.btnSave')}
          </button>
        </div>
      </div>

      <style>{`
        .edit-form-input:focus {
          border-color: var(--accent) !important;
        }
        .edit-modal-close:active, .edit-cancel-btn:active, .edit-save-btn:active {
          transform: scale(var(--scale-button-active));
          transition: transform var(--duration-press) var(--ease-snappy);
        }
        @media (hover: hover) {
          .edit-modal-close:hover {
            color: var(--fg);
          }
          .edit-cancel-btn:hover {
            border-color: var(--fg);
            color: var(--fg);
          }
          .edit-save-btn:hover:not(:disabled) {
            opacity: 0.9;
          }
        }
      `}</style>
    </div>
  )
}
