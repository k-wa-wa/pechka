'use client'

import { useState, useEffect, useCallback } from 'react'
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

  const next = useCallback(() => {
    if (items.length <= 1) return
    setCurrent((c) => (c + 1) % items.length)
  }, [items.length])

  const prev = useCallback(() => {
    if (items.length <= 1) return
    setCurrent((c) => (c - 1 + items.length) % items.length)
  }, [items.length])

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
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
      style={{ position: 'relative', width: '100%', overflow: 'hidden' }}
    >
      {/* Main slide */}
      <Link
        href={`/contents/${item.short_id}`}
        style={{ display: 'block', position: 'relative' }}
      >
        <div
          style={{
            width: '100%',
            aspectRatio: '21/9',
            backgroundColor: '#0d1117',
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
                {slideItem.thumbnail_key ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={`/thumbnails/${slideItem.thumbnail_key}`}
                    alt={slideItem.title}
                    style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                  />
                ) : (
                  <div
                    style={{
                      width: '100%',
                      height: '100%',
                      background: 'linear-gradient(135deg, #161b22 0%, #0d1117 100%)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="#30363d" strokeWidth="1">
                      <polygon points="5 3 19 12 5 21 5 3" />
                    </svg>
                  </div>
                )}
              </div>
            )
          })}

          {/* Gradient overlay */}
          <div
            style={{
              position: 'absolute',
              inset: 0,
              background:
                'linear-gradient(to top, rgba(13,17,23,0.95) 0%, rgba(13,17,23,0.3) 50%, transparent 100%)',
              pointerEvents: 'none',
            }}
          />

          {/* Progress bar at the top of carousel */}
          {items.length > 1 && isPlaying && !isHovered && (
            <div
              key={current}
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                height: 3,
                backgroundColor: '#58a6ff',
                width: '100%',
                animation: `carousel-progress ${AUTO_SLIDE_INTERVAL}ms linear`,
                transformOrigin: 'left',
                zIndex: 10,
              }}
            />
          )}

          {/* Content info overlay */}
          <div
            style={{
              position: 'absolute',
              bottom: 0,
              left: 0,
              right: 0,
              padding: '24px 32px',
              zIndex: 2,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
              <span
                style={{
                  fontSize: 11,
                  padding: '2px 8px',
                  borderRadius: 4,
                  backgroundColor: 'rgba(31,111,235,0.85)',
                  color: '#58a6ff',
                }}
              >
                {CONTENT_TYPE_LABEL[item.content_type]}
              </span>
              {item.duration_seconds != null && (
                <span style={{ fontSize: 13, color: '#8b949e' }}>
                  {formatDuration(item.duration_seconds)}
                </span>
              )}
            </div>
            <h2
              style={{
                margin: 0,
                fontSize: 'clamp(16px, 3vw, 24px)',
                fontWeight: 700,
                color: '#e6edf3',
                lineHeight: 1.3,
                textShadow: '0 1px 4px rgba(0,0,0,0.5)',
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
                      borderRadius: 4,
                      backgroundColor: '#1f6feb22',
                      color: '#58a6ff',
                      border: '1px solid #1f6feb44',
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
            onClick={(e) => {
              e.preventDefault()
              prev()
            }}
            style={{
              position: 'absolute',
              left: 12,
              top: '50%',
              transform: 'translateY(-50%)',
              backgroundColor: 'rgba(22,27,34,0.8)',
              border: '1px solid #30363d',
              borderRadius: '50%',
              width: 36,
              height: 36,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: 'pointer',
              color: '#e6edf3',
              zIndex: 3,
            }}
            aria-label="Previous"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <polyline points="15 18 9 12 15 6" />
            </svg>
          </button>
          <button
            onClick={(e) => {
              e.preventDefault()
              next()
            }}
            style={{
              position: 'absolute',
              right: 12,
              top: '50%',
              transform: 'translateY(-50%)',
              backgroundColor: 'rgba(22,27,34,0.8)',
              border: '1px solid #30363d',
              borderRadius: '50%',
              width: 36,
              height: 36,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: 'pointer',
              color: '#e6edf3',
              zIndex: 3,
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
              style={{
                backgroundColor: 'rgba(22,27,34,0.8)',
                border: '1px solid #30363d',
                borderRadius: '50%',
                width: 24,
                height: 24,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                cursor: 'pointer',
                color: '#8b949e',
                padding: 0,
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
              {items.map((_, i) => (
                <button
                  key={i}
                  onClick={(e) => {
                    e.preventDefault()
                    setCurrent(i)
                  }}
                  style={{
                    width: i === current ? 20 : 6,
                    height: 6,
                    borderRadius: 3,
                    backgroundColor: i === current ? '#58a6ff' : '#30363d',
                    border: 'none',
                    cursor: 'pointer',
                    padding: 0,
                    transition: 'width 0.2s, background 0.2s',
                  }}
                  aria-label={`Slide ${i + 1}`}
                />
              ))}
            </div>
          </div>
        </>
      )}

      <style jsx global>{`
        @keyframes carousel-progress {
          from {
            transform: scaleX(0);
          }
          to {
            transform: scaleX(1);
          }
        }
      `}</style>
    </div>
  )
}

