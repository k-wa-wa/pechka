'use client'

import { useRouter } from 'next/navigation'
import type { ContentType } from '@/lib/types'

interface FilterOption {
  value: ContentType | ''
  label: string
}

interface Props {
  types: FilterOption[]
  currentType: string
}

export default function FilterBar({ types, currentType }: Props) {
  const router = useRouter()

  function handleChange(value: string) {
    if (value) {
      router.push(`/?type=${value}`)
    } else {
      router.push('/')
    }
  }

  return (
    <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
      {types.map((t) => {
        const active = t.value === currentType
        return (
          <button
            key={t.value}
            onClick={() => handleChange(t.value)}
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
            {t.label}
          </button>
        )
      })}
    </div>
  )
}
