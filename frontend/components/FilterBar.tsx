'use client'

import { useRouter } from 'next/navigation'
import type { ContentType } from '@/lib/types'
import { useLanguage } from '@/lib/i18n/LanguageContext'

export interface FilterOption {
  value: ContentType | ''
  labelKey?: string
  label?: string
}

interface Props {
  types: FilterOption[]
  currentType: string
}

export default function FilterBar({ types, currentType }: Props) {
  const router = useRouter()
  const { t } = useLanguage()

  function handleChange(value: string) {
    if (value) {
      router.push(`/?type=${value}`)
    } else {
      router.push('/')
    }
  }

  return (
    <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
      {types.map((tOpt) => {
        const active = tOpt.value === currentType
        const label = tOpt.labelKey ? t(tOpt.labelKey) : (tOpt.label ?? '')
        return (
          <button
            key={tOpt.value}
            onClick={() => handleChange(tOpt.value)}
            style={{
              padding: '5px 12px',
              borderRadius: 20,
              border: '1px solid',
              borderColor: active ? '#58a6ff' : '#30363d',
              backgroundColor: active ? '#1f6feb33' : 'transparent',
              color: active ? '#58a6ff' : '#8b949e',
              cursor: 'pointer',
              fontSize: 13,
              transition: 'all 0.15s',
            }}
          >
            {label}
          </button>
        )
      })}
    </div>
  )
}
