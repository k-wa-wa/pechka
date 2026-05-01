'use client'

import { useEffect, useRef, useState } from 'react'
import type { MongoVariant } from '@/lib/types'

interface Props {
  variants: MongoVariant[]
}

const QUALITY_LABELS: Record<string, string> = {
  master: '自動',
  '1080p': '1080p',
  '720p': '720p',
  '480p': '480p',
  audio: 'Audio',
  original: 'Original',
}

export default function VideoPlayer({ variants }: Props) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const hlsRef = useRef<import('hls.js').default | null>(null)
  const [selectedVariant, setSelectedVariant] = useState<string>('master')
  const [error, setError] = useState<string | null>(null)

  // Find available quality variants (exclude master for manual selection)
  const qualityVariants = variants.filter((v) =>
    ['1080p', '720p', '480p', 'audio', 'original'].includes(v.variant_type)
  )
  const masterVariant = variants.find((v) => v.variant_type === 'master')

  // Get currently selected variant object
  const currentVariant =
    selectedVariant === 'master'
      ? masterVariant
      : variants.find((v) => v.variant_type === selectedVariant)

  useEffect(() => {
    if (!videoRef.current || !currentVariant) return

    const video = videoRef.current
    const src = `/${currentVariant.hls_key}`

    let destroyed = false

    async function init() {
      const Hls = (await import('hls.js')).default

      if (destroyed) return

      if (hlsRef.current) {
        hlsRef.current.destroy()
        hlsRef.current = null
      }

      if (Hls.isSupported()) {
        const hls = new Hls({
          enableWorker: true,
        })
        hlsRef.current = hls
        hls.loadSource(src)
        hls.attachMedia(video)
        hls.on(Hls.Events.ERROR, (_event, data) => {
          if (data.fatal) {
            setError(`HLS error: ${data.type}`)
          }
        })
        hls.on(Hls.Events.MANIFEST_PARSED, () => {
          video.play().catch(() => {
            // autoplay blocked — ignore
          })
        })
      } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
        // Safari native HLS
        video.src = src
        video.play().catch(() => {})
      } else {
        setError('HLS is not supported in this browser.')
      }
    }

    init().catch((e: unknown) => {
      if (!destroyed) setError(String(e))
    })

    return () => {
      destroyed = true
      if (hlsRef.current) {
        hlsRef.current.destroy()
        hlsRef.current = null
      }
    }
  }, [currentVariant])

  if (!masterVariant && variants.length === 0) {
    return (
      <div
        style={{
          aspectRatio: '16/9',
          backgroundColor: '#0d1117',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: '#8b949e',
          borderRadius: 8,
        }}
      >
        動画が見つかりません
      </div>
    )
  }

  return (
    <div style={{ position: 'relative', width: '100%' }}>
      <div
        style={{
          position: 'relative',
          width: '100%',
          aspectRatio: '16/9',
          backgroundColor: '#000',
          borderRadius: 8,
          overflow: 'hidden',
        }}
      >
        <video
          ref={videoRef}
          controls
          playsInline
          style={{ width: '100%', height: '100%', display: 'block' }}
        />
        {error && (
          <div
            style={{
              position: 'absolute',
              inset: 0,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              backgroundColor: 'rgba(0,0,0,0.7)',
              color: '#ff7b72',
              fontSize: 14,
              textAlign: 'center',
              padding: 16,
            }}
          >
            {error}
          </div>
        )}
      </div>

      {/* Quality selector */}
      {(masterVariant || qualityVariants.length > 0) && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            marginTop: 8,
          }}
        >
          <span style={{ fontSize: 13, color: '#8b949e' }}>画質:</span>
          <div style={{ display: 'flex', gap: 4 }}>
            {masterVariant && (
              <button
                onClick={() => setSelectedVariant('master')}
                style={{
                  padding: '4px 10px',
                  borderRadius: 4,
                  border: '1px solid',
                  borderColor: selectedVariant === 'master' ? '#58a6ff' : '#30363d',
                  backgroundColor:
                    selectedVariant === 'master' ? '#1f6feb33' : 'transparent',
                  color: selectedVariant === 'master' ? '#58a6ff' : '#8b949e',
                  cursor: 'pointer',
                  fontSize: 12,
                }}
              >
                自動
              </button>
            )}
            {qualityVariants.map((v) => (
              <button
                key={v.variant_type}
                onClick={() => setSelectedVariant(v.variant_type)}
                style={{
                  padding: '4px 10px',
                  borderRadius: 4,
                  border: '1px solid',
                  borderColor:
                    selectedVariant === v.variant_type ? '#58a6ff' : '#30363d',
                  backgroundColor:
                    selectedVariant === v.variant_type
                      ? '#1f6feb33'
                      : 'transparent',
                  color:
                    selectedVariant === v.variant_type ? '#58a6ff' : '#8b949e',
                  cursor: 'pointer',
                  fontSize: 12,
                }}
              >
                {QUALITY_LABELS[v.variant_type] ?? v.variant_type}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
