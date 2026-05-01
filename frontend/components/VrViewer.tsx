'use client'

import { useEffect, useRef } from 'react'
import type { MongoVariant } from '@/lib/types'

interface Props {
  variants: MongoVariant[]
}

export default function VrViewer({ variants }: Props) {
  const containerRef = useRef<HTMLDivElement>(null)
  const initializedRef = useRef(false)

  const masterVariant =
    variants.find((v) => v.variant_type === 'master') ??
    variants.find((v) => v.variant_type === 'original') ??
    variants[0]

  useEffect(() => {
    if (!masterVariant || initializedRef.current) return
    initializedRef.current = true

    async function init() {
      await import('aframe')

      if (!containerRef.current) return

      const src = `/${masterVariant!.hls_key}`

      const scene = document.createElement('a-scene')
      scene.setAttribute('embedded', '')
      scene.setAttribute('style', 'width:100%;height:100%;')
      scene.setAttribute('vr-mode-ui', 'enabled: true')
      scene.setAttribute('loading-screen', 'dotsColor: #58a6ff; backgroundColor: #0d1117')

      const assets = document.createElement('a-assets')
      const video = document.createElement('video')
      video.id = 'vr-video'
      video.setAttribute('src', src)
      video.setAttribute('crossorigin', 'anonymous')
      video.setAttribute('loop', 'true')
      video.setAttribute('preload', 'auto')
      assets.appendChild(video)
      scene.appendChild(assets)

      const sphere = document.createElement('a-videosphere')
      sphere.setAttribute('src', '#vr-video')
      sphere.setAttribute('rotation', '0 -90 0')
      scene.appendChild(sphere)

      const camera = document.createElement('a-entity')
      camera.setAttribute('camera', '')
      camera.setAttribute('look-controls', '')
      camera.setAttribute('wasd-controls', 'enabled: false')
      scene.appendChild(camera)

      containerRef.current.appendChild(scene)

      // Auto-play after scene loads
      scene.addEventListener('loaded', () => {
        video.play().catch(() => {})
      })
    }

    init().catch(console.error)

    return () => {
      // Cleanup: remove a-scene if it was added
      if (containerRef.current) {
        const scene = containerRef.current.querySelector('a-scene')
        if (scene) scene.remove()
      }
      initializedRef.current = false
    }
  }, [masterVariant])

  if (!masterVariant) {
    return (
      <div
        style={{
          width: '100%',
          height: '100vh',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          backgroundColor: '#0d1117',
          color: '#8b949e',
        }}
      >
        VRコンテンツが見つかりません
      </div>
    )
  }

  return (
    <div
      ref={containerRef}
      style={{
        width: '100%',
        height: 'calc(100vh - 60px)',
        backgroundColor: '#0d1117',
        position: 'relative',
      }}
    />
  )
}
