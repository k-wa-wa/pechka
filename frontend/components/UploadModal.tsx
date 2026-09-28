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
            {t('uploadModal.title')}
          </h2>
          <button
            onClick={onClose}
            disabled={uploading}
            className="upload-modal-close"
            style={{
              background: 'none',
              border: 'none',
              color: 'var(--muted)',
              cursor: uploading ? 'not-allowed' : 'pointer',
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
            <span style={{ fontSize: 13, color: 'var(--muted)' }}>{t('uploadModal.fieldFile')}</span>
            <input
              type="file"
              accept="video/*"
              disabled={uploading}
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
              className="upload-form-input"
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
            <span style={{ fontSize: 13, color: 'var(--muted)' }}>{t('uploadModal.fieldTitle')}</span>
            <input
              value={title}
              disabled={uploading}
              onChange={(e) => setTitle(e.target.value)}
              className="upload-form-input"
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
            <span style={{ fontSize: 13, color: 'var(--muted)' }}>
              {t('uploadModal.fieldDescription')}
            </span>
            <textarea
              value={description}
              disabled={uploading}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className="upload-form-input"
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
              {t('uploadModal.fieldTags')}
            </span>
            <input
              value={tags}
              disabled={uploading}
              onChange={(e) => setTags(e.target.value)}
              placeholder="tag1, tag2, tag3"
              className="upload-form-input"
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

          {uploading && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              <div
                style={{
                  height: 6,
                  borderRadius: 3,
                  backgroundColor: 'var(--line)',
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    height: '100%',
                    width: `${progress}%`,
                    backgroundColor: 'var(--accent)',
                    transition: 'width 0.2s',
                  }}
                />
              </div>
              <span style={{ fontSize: 12, color: 'var(--muted)' }}>
                {t('uploadModal.uploading')} {progress}%
              </span>
            </div>
          )}

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
            disabled={uploading}
            className="upload-cancel-btn"
            style={{
              padding: '8px 16px',
              borderRadius: 'var(--radius-md)',
              border: '1px solid var(--line)',
              backgroundColor: 'transparent',
              color: 'var(--muted)',
              cursor: uploading ? 'not-allowed' : 'pointer',
              fontSize: 14,
              transition: 'border-color 0.15s, color 0.15s, transform var(--duration-release) var(--ease-spring)',
            }}
          >
            {t('uploadModal.btnCancel')}
          </button>
          <button
            onClick={handleUpload}
            disabled={!canUpload}
            className="upload-submit-btn"
            style={{
              padding: '8px 16px',
              borderRadius: 'var(--radius-md)',
              border: 'none',
              backgroundColor: canUpload ? 'var(--accent)' : 'var(--accent-emphasis)',
              color: canUpload ? '#191816' : 'var(--muted)',
              cursor: canUpload ? 'pointer' : 'not-allowed',
              fontSize: 14,
              fontWeight: 600,
              transition: 'opacity 0.15s, transform var(--duration-release) var(--ease-spring)',
            }}
          >
            {uploading ? t('uploadModal.uploading') : t('uploadModal.btnUpload')}
          </button>
        </div>
      </div>

      <style>{`
        .upload-form-input:focus {
          border-color: var(--accent) !important;
        }
        .upload-modal-close:active, .upload-cancel-btn:active, .upload-submit-btn:active {
          transform: scale(var(--scale-button-active));
          transition: transform var(--duration-press) var(--ease-snappy);
        }
        @media (hover: hover) {
          .upload-modal-close:hover {
            color: var(--fg);
          }
          .upload-cancel-btn:hover {
            border-color: var(--fg);
            color: var(--fg);
          }
          .upload-submit-btn:hover:not(:disabled) {
            opacity: 0.9;
          }
        }
      `}</style>
    </div>
  )
}
