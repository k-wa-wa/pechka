'use client'

import dynamic from 'next/dynamic'
import type { MongoVariant } from '@/lib/types'
import { useLanguage } from '@/lib/i18n/LanguageContext'

function LoadingFallback({ textKey }: { textKey: string }) {
  const { t } = useLanguage()
  return (
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
      {t(textKey)}
    </div>
  )
}

const VideoPlayer = dynamic(() => import('./VideoPlayer'), {
  ssr: false,
  loading: () => <LoadingFallback textKey="player.loading" />,
})

const VrViewer = dynamic(() => import('./VrViewer'), {
  ssr: false,
  loading: () => <LoadingFallback textKey="player.loadingVr" />,
})

interface Props {
  variants: MongoVariant[]
  isVr: boolean
  shortId: string
  hasSubtitles: boolean
}

export default function ContentPlayer({ variants, isVr, shortId, hasSubtitles }: Props) {
  if (isVr) {
    return <VrViewer variants={variants} />
  }
  return <VideoPlayer variants={variants} shortId={shortId} hasSubtitles={hasSubtitles} />
}
