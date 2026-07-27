'use client'

import { useEffect, useRef, useState } from 'react'
import type { MongoVariant } from '@/lib/types'
import { useLanguage } from '@/lib/i18n/LanguageContext'

interface Props {
  variants: MongoVariant[]
  shortId: string
  hasSubtitles: boolean
}

const QUALITY_LABELS: Record<string, string> = {
  master: 'Auto',
  '1080p': '1080p',
  '720p': '720p',
  '480p': '480p',
  audio: 'Audio',
  original: 'Original',
}

export default function VideoPlayer({ variants, shortId, hasSubtitles }: Props) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const trackRef = useRef<HTMLTrackElement>(null)
  const hlsRef = useRef<import('hls.js').default | null>(null)
  const [selectedVariant, setSelectedVariant] = useState<string>('master')
  const [error, setError] = useState<string | null>(null)
  const [trackReloadKey, setTrackReloadKey] = useState(0)
  const { t } = useLanguage()

  const lastPlaybackState = useRef({ currentTime: 0, paused: false })

  const handleVariantChange = (variantType: string) => {
    if (videoRef.current) {
      lastPlaybackState.current = {
        currentTime: videoRef.current.currentTime,
        paused: videoRef.current.paused,
      }
    }
    setSelectedVariant(variantType)
  }

  // Find available quality variants (exclude master for manual selection) and sort by quality (best to worst)
  const QUALITY_ORDER = ['1080p', '720p', '480p', 'audio', 'original']
  const qualityVariants = variants
    .filter((v) => QUALITY_ORDER.includes(v.variant_type))
    .sort((a, b) => {
      const aIndex = QUALITY_ORDER.indexOf(a.variant_type)
      const bIndex = QUALITY_ORDER.indexOf(b.variant_type)
      return aIndex - bIndex
    })
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

    // Retrieve and reset the captured playback state
    const { currentTime: prevTime, paused: prevPaused } = lastPlaybackState.current
    lastPlaybackState.current = { currentTime: 0, paused: true }

    let destroyed = false
    let onLoadedMetadata: (() => void) | null = null

    // hls.js の attachMedia() が video 要素にMediaSourceをアタッチする際、
    // 既に存在する <track> のcue読み込み状態を壊す(readyState はLOADEDになるがcuesが0件になる)
    // ことがある。MANIFEST_PARSED時点ではまだ早すぎて直らなかったため、実際に再生が始まった
    // (playing イベント)タイミングで key を変更し、Reactに<track>要素を作り直させる。
    function reloadTrack() {
      setTrackReloadKey((k) => k + 1)
    }
    video.addEventListener('playing', reloadTrack, { once: true })

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
          startPosition: prevTime,
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
          if (!prevPaused) {
            video.play().catch(() => {
              // autoplay blocked — ignore
            })
          }
        })
      } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
        // Safari native HLS
        video.src = src
        onLoadedMetadata = () => {
          if (prevTime > 0) {
            video.currentTime = prevTime
          }
          if (!prevPaused) {
            video.play().catch(() => { })
          }
        }
        video.addEventListener('loadedmetadata', onLoadedMetadata)
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
      if (onLoadedMetadata) {
        video.removeEventListener('loadedmetadata', onLoadedMetadata)
      }
      video.removeEventListener('playing', reloadTrack)
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
        {t('player.notFound')}
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
        >
          {hasSubtitles && (
            <track
              key={trackReloadKey}
              ref={trackRef}
              kind="subtitles"
              srcLang="ja"
              label="日本語"
              src={`/api/v1/contents/${shortId}/subtitles/ja`}
              default
            />
          )}
        </video>
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
          <span style={{ fontSize: 13, color: '#8b949e' }}>{t('player.quality')}</span>
          <div style={{ display: 'flex', gap: 4 }}>
            {masterVariant && (
              <button
                onClick={() => handleVariantChange('master')}
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
                Auto
              </button>
            )}
            {qualityVariants.map((v) => (
              <button
                key={v.variant_type}
                onClick={() => handleVariantChange(v.variant_type)}
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
