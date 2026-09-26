'use client'

import { useState } from 'react'
import type { Content } from '@/lib/types'
import { uploadContent } from '@/lib/api'
import { useLanguage } from '@/lib/i18n/LanguageContext'

interface Props {
  onClose: () => void
  onUploaded: (content: Content) => void
}

export default function UploadModal({ onClose, onUploaded }: Props) {
  const [file, setFile] = useState<File | null>(null)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [tags, setTags] = useState('')
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const { t } = useLanguage()

  const canUpload = file != null && title.trim() !== '' && !uploading

  async function handleUpload() {
    if (!file) return
    setUploading(true)
    setProgress(0)
    setError(null)
    try {
      const res = await uploadContent(
        file,
        {
          title,
          description: description || undefined,
          tags: tags
            .split(',')
            .map((tag) => tag.trim())
            .filter(Boolean),
        },
        setProgress
      )
      onUploaded(res.content)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('uploadModal.uploadFailed'))
    } finally {
      setUploading(false)
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
      onClick={uploading ? undefined : onClose}
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
            {t('uploadModal.title')}
          </h2>
          <button
            onClick={onClose}
            disabled={uploading}
            style={{
              background: 'none',
              border: 'none',
              color: '#8b949e',
              cursor: uploading ? 'not-allowed' : 'pointer',
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
            <span style={{ fontSize: 13, color: '#8b949e' }}>{t('uploadModal.fieldFile')}</span>
            <input
              type="file"
              accept="video/*"
              disabled={uploading}
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
              style={{
                backgroundColor: '#0d1117',
                border: '1px solid #30363d',
                borderRadius: 6,
                color: '#e6edf3',
                padding: '8px 12px',
                fontSize: 14,
                outline: 'none',
              }}
            />
          </label>

          <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <span style={{ fontSize: 13, color: '#8b949e' }}>{t('uploadModal.fieldTitle')}</span>
            <input
              value={title}
              disabled={uploading}
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
            <span style={{ fontSize: 13, color: '#8b949e' }}>
              {t('uploadModal.fieldDescription')}
            </span>
            <textarea
              value={description}
              disabled={uploading}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
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
              {t('uploadModal.fieldTags')}
            </span>
            <input
              value={tags}
              disabled={uploading}
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

          {uploading && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              <div
                style={{
                  height: 6,
                  borderRadius: 3,
                  backgroundColor: '#30363d',
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    height: '100%',
                    width: `${progress}%`,
                    backgroundColor: '#1f6feb',
                    transition: 'width 0.2s',
                  }}
                />
              </div>
              <span style={{ fontSize: 12, color: '#8b949e' }}>
                {t('uploadModal.uploading')} {progress}%
              </span>
            </div>
          )}

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
            disabled={uploading}
            style={{
              padding: '8px 16px',
              borderRadius: 6,
              border: '1px solid #30363d',
              backgroundColor: 'transparent',
              color: '#8b949e',
              cursor: uploading ? 'not-allowed' : 'pointer',
              fontSize: 14,
            }}
          >
            {t('uploadModal.btnCancel')}
          </button>
          <button
            onClick={handleUpload}
            disabled={!canUpload}
            style={{
              padding: '8px 16px',
              borderRadius: 6,
              border: 'none',
              backgroundColor: canUpload ? '#1f6feb' : '#1f6feb88',
              color: '#e6edf3',
              cursor: canUpload ? 'pointer' : 'not-allowed',
              fontSize: 14,
              fontWeight: 600,
            }}
          >
            {uploading ? t('uploadModal.uploading') : t('uploadModal.btnUpload')}
          </button>
        </div>
      </div>
    </div>
  )
}
