'use client'

import dynamic from 'next/dynamic'
import type { MongoVariant } from '@/lib/types'

const VideoPlayer = dynamic(() => import('./VideoPlayer'), {
  ssr: false,
  loading: () => (
    <div
      style={{
        width: '100%',
        aspectRatio: '16/9',
        backgroundColor: '#0d1117',
        borderRadius: 8,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        color: '#8b949e',
      }}
    >
      読み込み中...
    </div>
  ),
})

const VrViewer = dynamic(() => import('./VrViewer'), {
  ssr: false,
  loading: () => (
    <div
      style={{
        width: '100%',
        height: 'calc(100vh - 60px)',
        backgroundColor: '#0d1117',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        color: '#8b949e',
      }}
    >
      VRビューアを読み込み中...
    </div>
  ),
})

interface Props {
  variants: MongoVariant[]
  isVr: boolean
}

export default function ContentPlayer({ variants, isVr }: Props) {
  if (isVr) {
    return <VrViewer variants={variants} />
  }
  return <VideoPlayer variants={variants} />
}
