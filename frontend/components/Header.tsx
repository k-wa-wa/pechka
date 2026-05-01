'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import SearchModal from './SearchModal'

export default function Header() {
  const [searchOpen, setSearchOpen] = useState(false)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const pathname = usePathname()

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
    { href: '/', label: 'Home' },
    { href: '/admin', label: 'Admin' },
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
          title="Search (Cmd+K)"
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
          <span className="search-hint">Search</span>
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
    </>
  )
}
