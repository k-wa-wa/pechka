'use client'

import { useEffect } from 'react'
import { useLanguage, Language } from '@/lib/i18n/LanguageContext'

interface Props {
  isOpen: boolean
  onClose: () => void
}

export default function SettingsModal({ isOpen, onClose }: Props) {
  const { language, setLanguage, t } = useLanguage()

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, onClose])

  if (!isOpen) return null

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        backgroundColor: 'rgba(0, 0, 0, 0.6)',
        backdropFilter: 'blur(4px)',
        zIndex: 200,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 16,
      }}
      onClick={onClose}
    >
      <div
        style={{
          backgroundColor: 'var(--card)',
          border: '1px solid var(--line)',
          borderRadius: 'var(--radius-lg)',
          width: '100%',
          maxWidth: 480,
          boxShadow: '0 12px 32px rgba(0, 0, 0, 0.5)',
          overflow: 'hidden',
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
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="var(--accent)"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.38a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z" />
              <circle cx="12" cy="12" r="3" />
            </svg>
            <h2 style={{ fontSize: 16, fontWeight: 600, color: 'var(--fg)', margin: 0 }}>
              {t('settings.title')}
            </h2>
          </div>
          <button
            onClick={onClose}
            className="settings-close-btn"
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
            aria-label={t('settings.close')}
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div style={{ padding: 20 }}>
          <div style={{ marginBottom: 16 }}>
            <label
              style={{
                display: 'block',
                fontSize: 14,
                fontWeight: 500,
                color: 'var(--muted)',
                marginBottom: 8,
              }}
            >
              {t('settings.language')}
            </label>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {[
                { code: 'ja' as Language, label: t('settings.langJa') },
                { code: 'en' as Language, label: t('settings.langEn') },
              ].map((item) => (
                <button
                  key={item.code}
                  onClick={() => setLanguage(item.code)}
                  className="settings-lang-btn"
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '10px 14px',
                    borderRadius: 'var(--radius-md)',
                    border: `1px solid ${language === item.code ? 'var(--accent)' : 'var(--line)'}`,
                    backgroundColor: language === item.code ? 'var(--accent-emphasis)' : 'var(--bg)',
                    color: 'var(--fg)',
                    fontSize: 14,
                    fontWeight: language === item.code ? 600 : 400,
                    cursor: 'pointer',
                    transition: 'all 0.15s ease, transform var(--duration-release) var(--ease-spring)',
                  }}
                >
                  <span>{item.label}</span>
                  {language === item.code && (
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" strokeWidth="2.5">
                      <polyline points="20 6 9 17 4 12" />
                    </svg>
                  )}
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div
          style={{
            padding: '12px 20px',
            borderTop: '1px solid var(--line)',
            backgroundColor: 'var(--bg)',
            display: 'flex',
            justifyContent: 'flex-end',
          }}
        >
          <button
            onClick={onClose}
            className="settings-close-action-btn"
            style={{
              padding: '6px 16px',
              borderRadius: 'var(--radius-md)',
              fontSize: 14,
              fontWeight: 500,
              backgroundColor: 'var(--card)',
              border: '1px solid var(--line)',
              color: 'var(--fg)',
              cursor: 'pointer',
              transition: 'background-color 0.15s, border-color 0.15s, transform var(--duration-release) var(--ease-spring)',
            }}
          >
            {t('settings.close')}
          </button>
        </div>
      </div>

      <style>{`
        .settings-close-btn:active, .settings-lang-btn:active, .settings-close-action-btn:active {
          transform: scale(var(--scale-button-active));
          transition: transform var(--duration-press) var(--ease-snappy);
        }
        @media (hover: hover) {
          .settings-close-btn:hover {
            color: var(--fg);
          }
          .settings-close-action-btn:hover {
            border-color: var(--accent);
          }
        }
      `}</style>
    </div>
  )
}
