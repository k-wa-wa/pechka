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
            className="filter-chip"
            style={{
              padding: '5px 12px',
              borderRadius: 'var(--radius-full)',
              border: '1px solid',
              borderColor: active ? 'var(--accent)' : 'var(--line)',
              backgroundColor: active ? 'var(--accent-emphasis)' : 'transparent',
              color: active ? 'var(--accent)' : 'var(--muted)',
              cursor: 'pointer',
              fontSize: 13,
              transition: 'all 0.15s, transform var(--duration-release) var(--ease-spring)',
            }}
          >
            {label}
          </button>
        )
      })}
      <style>{`
        .filter-chip:active {
          transform: scale(var(--scale-button-active));
          transition: transform var(--duration-press) var(--ease-snappy);
        }
        @media (hover: hover) {
          .filter-chip:not(:active):hover {
            border-color: var(--accent);
            color: var(--fg);
          }
        }
      `}</style>
    </div>
  )
}
