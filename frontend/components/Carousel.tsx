'use client'

import { useState, useEffect, useCallback, useRef } from 'react'
import Link from 'next/link'
import type { MongoContent, ContentType } from '@/lib/types'

const CONTENT_TYPE_LABEL: Record<ContentType, string> = {
  video: 'Video',
  image_gallery: 'Gallery',
  vr360: 'VR360',
  document: 'Document',
}

const AUTO_SLIDE_INTERVAL = 5000

function formatDuration(seconds: number): string {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  return `${m}:${String(s).padStart(2, '0')}`
}

interface Props {
  items: MongoContent[]
}

export default function Carousel({ items }: Props) {
  const [current, setCurrent] = useState(0)
  const [isPlaying, setIsPlaying] = useState(true)
  const [isHovered, setIsHovered] = useState(false)
  const touchStartX = useRef<number | null>(null)
  const touchStartY = useRef<number | null>(null)
  const isSwiping = useRef(false)

  const next = useCallback(() => {
    if (items.length <= 1) return
    setCurrent((c) => (c + 1) % items.length)
  }, [items.length])

  const prev = useCallback(() => {
    if (items.length <= 1) return
    setCurrent((c) => (c - 1 + items.length) % items.length)
  }, [items.length])

  const handleTouchStart = (e: React.TouchEvent) => {
    touchStartX.current = e.touches[0].clientX
    touchStartY.current = e.touches[0].clientY
    isSwiping.current = false
  }

  const handleTouchMove = (e: React.TouchEvent) => {
    if (touchStartX.current === null || touchStartY.current === null) return
    const diffX = e.touches[0].clientX - touchStartX.current
    const diffY = e.touches[0].clientY - touchStartY.current
    if (Math.abs(diffX) > 10 && Math.abs(diffX) > Math.abs(diffY)) {
      isSwiping.current = true
    }
  }

  const handleTouchEnd = (e: React.TouchEvent) => {
    if (touchStartX.current === null) return
    const diffX = e.changedTouches[0].clientX - touchStartX.current
    if (Math.abs(diffX) > 35) {
      if (diffX > 0) {
        prev()
      } else {
        next()
      }
    }
    touchStartX.current = null
    touchStartY.current = null
    setTimeout(() => {
      isSwiping.current = false
    }, 120)
  }

  const handleLinkClick = (e: React.MouseEvent) => {
    if (isSwiping.current) {
      e.preventDefault()
    }
  }

  useEffect(() => {
    if (!isPlaying || isHovered || items.length <= 1) return

    const timer = setInterval(() => {
      next()
    }, AUTO_SLIDE_INTERVAL)

    return () => clearInterval(timer)
  }, [isPlaying, isHovered, items.length, next])

  if (items.length === 0) return null

  const item = items[current]

  return (
    <div
      onMouseEnter={() => {
        if (typeof window !== 'undefined' && window.matchMedia('(hover: hover)').matches) {
          setIsHovered(true)
        }
      }}
      onMouseLeave={() => setIsHovered(false)}
      onTouchStart={handleTouchStart}
      onTouchMove={handleTouchMove}
      onTouchEnd={handleTouchEnd}
      style={{ position: 'relative', width: '100%', overflow: 'hidden', touchAction: 'pan-y' }}
    >
      {/* Main slide */}
      <Link
        href={`/contents/${item.short_id}`}
        onClick={handleLinkClick}
        style={{ display: 'block', position: 'relative' }}
      >
        <div
          className="carousel-slide-box"
          style={{
            width: '100%',
            backgroundColor: 'var(--bg)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            overflow: 'hidden',
            position: 'relative',
          }}
        >
          {items.map((slideItem, index) => {
            const isActive = index === current
            return (
              <div
                key={slideItem.short_id || index}
                style={{
                  position: index === 0 ? 'relative' : 'absolute',
                  inset: 0,
                  opacity: isActive ? 1 : 0,
                  transition: 'opacity 0.6s ease-in-out',
                  pointerEvents: isActive ? 'auto' : 'none',
                  width: '100%',
                  height: '100%',
                }}
              >
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src={
                    slideItem.thumbnail_key
                      ? `/thumbnails/${slideItem.thumbnail_key}`
                      : `/images/placeholder-${slideItem.content_type}.svg`
                  }
                  alt={slideItem.title}
                  onError={(e) => {
                    const fallback = `/images/placeholder-${slideItem.content_type}.svg`
                    if (!e.currentTarget.src.endsWith(fallback)) {
                      e.currentTarget.src = fallback
                    }
                  }}
                  style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                />
              </div>
            )
          })}

          {/* Gradient overlay */}
          <div
            style={{
              position: 'absolute',
              inset: 0,
              background:
                'linear-gradient(to top, rgba(25,24,22,0.95) 0%, rgba(25,24,22,0.3) 50%, transparent 100%)',
              pointerEvents: 'none',
            }}
          />

          {/* Content info overlay */}
          <div
            className="carousel-content-info"
            style={{
              position: 'absolute',
              bottom: 0,
              left: 0,
              right: 0,
              zIndex: 2,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
              <span
                style={{
                  fontSize: 11,
                  padding: '2px 8px',
                  borderRadius: 'var(--radius-sm)',
                  backgroundColor: 'var(--accent-emphasis)',
                  color: 'var(--accent)',
                  border: '1px solid var(--accent-border)',
                }}
              >
                {CONTENT_TYPE_LABEL[item.content_type]}
              </span>
              {item.duration_seconds != null && (
                <span style={{ fontSize: 13, color: 'var(--muted)' }}>
                  {formatDuration(item.duration_seconds)}
                </span>
              )}
            </div>
            <h2
              style={{
                margin: 0,
                fontSize: 'clamp(16px, 3vw, 24px)',
                fontWeight: 700,
                color: 'var(--fg)',
                lineHeight: 1.3,
                textShadow: '0 1px 4px rgba(0,0,0,0.6)',
              }}
            >
              {item.title}
            </h2>
            {item.tags.length > 0 && (
              <div style={{ display: 'flex', gap: 6, marginTop: 8, flexWrap: 'wrap' }}>
                {item.tags.slice(0, 4).map((tag) => (
                  <span
                    key={tag}
                    style={{
                      fontSize: 11,
                      padding: '2px 6px',
                      borderRadius: 'var(--radius-sm)',
                      backgroundColor: 'var(--accent-subtle)',
                      color: 'var(--accent)',
                      border: '1px solid var(--accent-border)',
                    }}
                  >
                    {tag}
                  </span>
                ))}
              </div>
            )}
          </div>
        </div>
      </Link>

      {/* Navigation buttons & controls */}
      {items.length > 1 && (
        <>
          <button
            className="carousel-arrow"
            onClick={(e) => {
              e.preventDefault()
              prev()
            }}
            style={{
              position: 'absolute',
              left: 12,
              top: '50%',
              transform: 'translateY(-50%)',
              backgroundColor: 'rgba(33, 31, 28, 0.8)',
              border: '1px solid var(--line)',
              borderRadius: '50%',
              width: 36,
              height: 36,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: 'pointer',
              color: 'var(--fg)',
              zIndex: 3,
              transition: 'background-color 0.15s, border-color 0.15s, transform var(--duration-release) var(--ease-spring)',
            }}
            aria-label="Previous"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <polyline points="15 18 9 12 15 6" />
            </svg>
          </button>
          <button
            className="carousel-arrow"
            onClick={(e) => {
              e.preventDefault()
              next()
            }}
            style={{
              position: 'absolute',
              right: 12,
              top: '50%',
              transform: 'translateY(-50%)',
              backgroundColor: 'rgba(33, 31, 28, 0.8)',
              border: '1px solid var(--line)',
              borderRadius: '50%',
              width: 36,
              height: 36,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: 'pointer',
              color: 'var(--fg)',
              zIndex: 3,
              transition: 'background-color 0.15s, border-color 0.15s, transform var(--duration-release) var(--ease-spring)',
            }}
            aria-label="Next"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <polyline points="9 18 15 12 9 6" />
            </svg>
          </button>

          {/* Dots & Play/Pause controls */}
          <div
            style={{
              position: 'absolute',
              bottom: 12,
              right: 16,
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              zIndex: 3,
            }}
          >
            <button
              onClick={() => setIsPlaying((p) => !p)}
              className="carousel-ctrl-btn"
              style={{
                backgroundColor: 'rgba(33, 31, 28, 0.8)',
                border: '1px solid var(--line)',
                borderRadius: '50%',
                width: 24,
                height: 24,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                cursor: 'pointer',
                color: 'var(--muted)',
                padding: 0,
                transition: 'color 0.15s, border-color 0.15s, transform var(--duration-release) var(--ease-spring)',
              }}
              aria-label={isPlaying ? 'Pause slideshow' : 'Start slideshow'}
              title={isPlaying ? 'Pause auto-slide' : 'Start auto-slide'}
            >
              {isPlaying ? (
                <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
                  <rect x="6" y="4" width="4" height="16" />
                  <rect x="14" y="4" width="4" height="16" />
                </svg>
              ) : (
                <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
                  <polygon points="5 3 19 12 5 21 5 3" />
                </svg>
              )}
            </button>

            <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
              {items.map((_, i) => {
                const isActive = i === current
                return (
                  <button
                    key={i}
                    onClick={(e) => {
                      e.preventDefault()
                      setCurrent(i)
                    }}
                    style={{
                      position: 'relative',
                      width: isActive ? 28 : 6,
                      height: 6,
                      borderRadius: 3,
                      backgroundColor: 'var(--line)',
                      border: 'none',
                      cursor: 'pointer',
                      padding: 0,
                      overflow: 'hidden',
                      transition: 'width 0.25s ease',
                    }}
                    aria-label={`Slide ${i + 1}`}
                  >
                    {isActive && (
                      <div
                        key={`prog-${current}`}
                        style={{
                          position: 'absolute',
                          inset: 0,
                          backgroundColor: 'var(--accent)',
                          borderRadius: 3,
                          transformOrigin: 'left',
                          animation: `indicator-progress ${AUTO_SLIDE_INTERVAL}ms linear`,
                          animationPlayState: isPlaying && !isHovered ? 'running' : 'paused',
                        }}
                      />
                    )}
                  </button>
                )
              })}
            </div>
          </div>

          <style>{`
            .carousel-arrow:active, .carousel-ctrl-btn:active {
              transform: scale(var(--scale-button-active));
              transition: transform var(--duration-press) var(--ease-snappy);
            }
            @media (hover: hover) {
              .carousel-arrow:hover, .carousel-ctrl-btn:hover {
                border-color: var(--accent);
                color: var(--fg);
              }
            }
          `}</style>
        </>
      )}
    </div>
  )
}

