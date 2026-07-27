'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import SearchModal from './SearchModal'
import SettingsModal from './SettingsModal'
import { useLanguage } from '@/lib/i18n/LanguageContext'

export default function Header() {
  const [searchOpen, setSearchOpen] = useState(false)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const pathname = usePathname()
  const { t } = useLanguage()

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setSearchOpen((v) => !v)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  // Close drawer on route change
  useEffect(() => {
    setDrawerOpen(false)
  }, [pathname])

  const navLinks = [
    { href: '/', label: t('nav.home') },
    { href: '/admin', label: t('nav.admin') },
  ]

  return (
    <>
      <header
        style={{
          height: 60,
          backgroundColor: '#161b22',
          borderBottom: '1px solid #30363d',
          display: 'flex',
          alignItems: 'center',
          padding: '0 24px',
          position: 'sticky',
          top: 0,
          zIndex: 100,
          gap: 16,
        }}
      >
        {/* Logo */}
        <Link
          href="/"
          style={{
            fontSize: 20,
            fontWeight: 700,
            color: '#e6edf3',
            letterSpacing: '-0.5px',
          }}
        >
          pechka
        </Link>

        {/* Desktop nav */}
        <nav
          style={{ display: 'flex', gap: 4, marginLeft: 16 }}
          className="desktop-nav"
        >
          {navLinks.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              style={{
                padding: '6px 12px',
                borderRadius: 6,
                fontSize: 14,
                color: pathname === link.href ? '#e6edf3' : '#8b949e',
                backgroundColor:
                  pathname === link.href ? '#0d1117' : 'transparent',
                transition: 'color 0.15s, background 0.15s',
              }}
            >
              {link.label}
            </Link>
          ))}
        </nav>

        <div style={{ flex: 1 }} />

        {/* Search icon */}
        <button
          onClick={() => setSearchOpen(true)}
          title={t('header.searchHint')}
          style={{
            background: 'none',
            border: '1px solid #30363d',
            borderRadius: 6,
            color: '#8b949e',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            gap: 6,
            padding: '5px 10px',
            fontSize: 13,
          }}
        >
          <svg
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
          >
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.35-4.35" />
          </svg>
          <span className="search-hint">{t('header.search')}</span>
          <kbd
            style={{
              fontSize: 10,
              border: '1px solid #30363d',
              borderRadius: 3,
              padding: '1px 4px',
            }}
            className="search-hint"
          >
            ⌘K
          </kbd>
        </button>

        {/* Settings icon */}
        <button
          onClick={() => setSettingsOpen(true)}
          title={t('header.settings')}
          style={{
            background: 'none',
            border: '1px solid #30363d',
            borderRadius: 6,
            color: '#8b949e',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '6px',
            fontSize: 13,
            transition: 'color 0.15s, border-color 0.15s',
          }}
        >
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.38a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z" />
            <circle cx="12" cy="12" r="3" />
          </svg>
        </button>

        {/* Mobile hamburger */}
        <button
          onClick={() => setDrawerOpen((v) => !v)}
          className="hamburger"
          style={{
            background: 'none',
            border: 'none',
            color: '#e6edf3',
            cursor: 'pointer',
            display: 'none',
            padding: 4,
          }}
          aria-label="Menu"
        >
          <svg
            width="22"
            height="22"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
          >
            {drawerOpen ? (
              <path d="M18 6 6 18M6 6l12 12" />
            ) : (
              <>
                <line x1="4" y1="6" x2="20" y2="6" />
                <line x1="4" y1="12" x2="20" y2="12" />
                <line x1="4" y1="18" x2="20" y2="18" />
              </>
            )}
          </svg>
        </button>
      </header>

      {/* Mobile drawer */}
      {drawerOpen && (
        <div
          style={{
            position: 'fixed',
            top: 60,
            left: 0,
            right: 0,
            backgroundColor: '#161b22',
            borderBottom: '1px solid #30363d',
            zIndex: 99,
            padding: '12px 24px',
          }}
          className="mobile-drawer"
        >
          {navLinks.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              style={{
                display: 'block',
                padding: '10px 0',
                color: pathname === link.href ? '#58a6ff' : '#e6edf3',
                fontSize: 15,
                borderBottom: '1px solid #30363d',
              }}
            >
              {link.label}
            </Link>
          ))}
        </div>
      )}

      <style>{`
        @media (max-width: 640px) {
          .desktop-nav { display: none !important; }
          .search-hint { display: none !important; }
          .hamburger { display: flex !important; }
        }
      `}</style>

      <SearchModal isOpen={searchOpen} onClose={() => setSearchOpen(false)} />
      <SettingsModal
        isOpen={settingsOpen}
        onClose={() => setSettingsOpen(false)}
      />
    </>
  )
}

