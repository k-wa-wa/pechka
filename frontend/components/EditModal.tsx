'use client'

import { useState } from 'react'
import type { Content, ContentStatus, UpdateContentRequest } from '@/lib/types'

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
          backgroundColor: '#161b22',
          border: '1px solid #30363d',
          borderRadius: 12,
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
            borderBottom: '1px solid #30363d',
          }}
        >
          <h2 style={{ margin: 0, fontSize: 16, color: '#e6edf3' }}>
            コンテンツを編集
          </h2>
          <button
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              color: '#8b949e',
              cursor: 'pointer',
              padding: 4,
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
            <span style={{ fontSize: 13, color: '#8b949e' }}>タイトル</span>
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              style={{
                backgroundColor: '#0d1117',
                border: '1px solid #30363d',
                borderRadius: 6,
                color: '#e6edf3',
                padding: '8px 12px',
                fontSize: 14,
                outline: 'none',
              }}
              onFocus={(e) => (e.target.style.borderColor = '#58a6ff')}
              onBlur={(e) => (e.target.style.borderColor = '#30363d')}
            />
          </label>

          <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <span style={{ fontSize: 13, color: '#8b949e' }}>説明</span>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={4}
              style={{
                backgroundColor: '#0d1117',
                border: '1px solid #30363d',
                borderRadius: 6,
                color: '#e6edf3',
                padding: '8px 12px',
                fontSize: 14,
                outline: 'none',
                resize: 'vertical',
                fontFamily: 'inherit',
              }}
              onFocus={(e) => (e.target.style.borderColor = '#58a6ff')}
              onBlur={(e) => (e.target.style.borderColor = '#30363d')}
            />
          </label>

          <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <span style={{ fontSize: 13, color: '#8b949e' }}>
              タグ <span style={{ fontWeight: 400 }}>(カンマ区切り)</span>
            </span>
            <input
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              placeholder="tag1, tag2, tag3"
              style={{
                backgroundColor: '#0d1117',
                border: '1px solid #30363d',
                borderRadius: 6,
                color: '#e6edf3',
                padding: '8px 12px',
                fontSize: 14,
                outline: 'none',
              }}
              onFocus={(e) => (e.target.style.borderColor = '#58a6ff')}
              onBlur={(e) => (e.target.style.borderColor = '#30363d')}
            />
          </label>

          <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <span style={{ fontSize: 13, color: '#8b949e' }}>ステータス</span>
            <select
              value={status}
              onChange={(e) => setStatus(e.target.value as ContentStatus)}
              style={{
                backgroundColor: '#0d1117',
                border: '1px solid #30363d',
                borderRadius: 6,
                color: '#e6edf3',
                padding: '8px 12px',
                fontSize: 14,
                outline: 'none',
                cursor: 'pointer',
              }}
              onFocus={(e) => (e.target.style.borderColor = '#58a6ff')}
              onBlur={(e) => (e.target.style.borderColor = '#30363d')}
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
                backgroundColor: '#ff7b7222',
                border: '1px solid #ff7b7244',
                borderRadius: 6,
                padding: '8px 12px',
                color: '#ff7b72',
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
            borderTop: '1px solid #30363d',
          }}
        >
          <button
            onClick={onClose}
            disabled={saving}
            style={{
              padding: '8px 16px',
              borderRadius: 6,
              border: '1px solid #30363d',
              backgroundColor: 'transparent',
              color: '#8b949e',
              cursor: 'pointer',
              fontSize: 14,
            }}
          >
            キャンセル
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            style={{
              padding: '8px 16px',
              borderRadius: 6,
              border: 'none',
              backgroundColor: saving ? '#1f6feb88' : '#1f6feb',
              color: '#e6edf3',
              cursor: saving ? 'not-allowed' : 'pointer',
              fontSize: 14,
              fontWeight: 600,
            }}
          >
            {saving ? '保存中...' : '保存'}
          </button>
        </div>
      </div>
    </div>
  )
}
