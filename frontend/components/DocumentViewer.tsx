'use client'

import React, { useEffect, useState } from 'react'
import { resolveDocumentViewerKind } from '@/lib/types'
import { useLanguage } from '@/lib/i18n/LanguageContext'

interface Props {
  title?: string
  documentKey?: string | null
  documentMimeType?: string | null
  initialMarkdown?: string
}

function isSafeHref(href: string): boolean {
  const trimmed = href.trim()
  if (trimmed.startsWith('/') && !trimmed.startsWith('//')) {
    return true
  }
  try {
    const parsed = new URL(trimmed)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:'
  } catch {
    return false
  }
}

function renderInlineMarkdown(text: string): React.ReactNode[] {
  const nodes: React.ReactNode[] = []
  // Match inline code `...`, links [text](url), bold **...**, italic *...*
  const tokenRegex = /(`[^`]+`)|(\[[^\]]+\]\([^)]+\))|(\*\*[^*]+\*\*)|(\*[^*]+\*)/g
  let lastIndex = 0
  let match: RegExpExecArray | null
  let key = 0

  while ((match = tokenRegex.exec(text)) !== null) {
    if (match.index > lastIndex) {
      nodes.push(text.slice(lastIndex, match.index))
    }
    const token = match[0]
    if (token.startsWith('`') && token.endsWith('`')) {
      nodes.push(
        <code
          key={`inline-${key++}`}
          style={{
            padding: '2px 6px',
            borderRadius: 'var(--radius-sm)',
            backgroundColor: 'var(--bg)',
            border: '1px solid var(--line)',
            fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
            fontSize: '0.9em',
          }}
        >
          {token.slice(1, -1)}
        </code>
      )
    } else if (token.startsWith('[')) {
      const closeBracket = token.indexOf('](')
      const label = token.slice(1, closeBracket)
      const href = token.slice(closeBracket + 2, -1)
      if (isSafeHref(href)) {
        nodes.push(
          <a
            key={`inline-${key++}`}
            href={href.trim()}
            target="_blank"
            rel="noopener noreferrer"
            style={{ color: 'var(--accent)', textDecoration: 'underline' }}
          >
            {label}
          </a>
        )
      } else {
        nodes.push(label)
      }
    } else if (token.startsWith('**') && token.endsWith('**')) {
      nodes.push(<strong key={`inline-${key++}`}>{token.slice(2, -2)}</strong>)
    } else if (token.startsWith('*') && token.endsWith('*')) {
      nodes.push(<em key={`inline-${key++}`}>{token.slice(1, -1)}</em>)
    }
    lastIndex = tokenRegex.lastIndex
  }

  if (lastIndex < text.length) {
    nodes.push(text.slice(lastIndex))
  }
  return nodes
}

function CodeBlock({ lang, code }: { lang: string; code: string }) {
  const { t } = useLanguage()
  const [copied, setCopied] = useState(false)

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(code)
      setCopied(true)
      setTimeout(() => setCopied(false), 1800)
    } catch {
      // Clipboard API unavailable
    }
  }

  return (
    <div
      style={{
        margin: '16px 0',
        borderRadius: 'var(--radius-md)',
        border: '1px solid var(--line)',
        backgroundColor: 'var(--bg)',
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '6px 12px',
          borderBottom: '1px solid var(--line)',
          fontSize: 12,
          color: 'var(--muted)',
        }}
      >
        <span style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace' }}>
          {lang || 'text'}
        </span>
        <button
          type="button"
          onClick={handleCopy}
          style={{
            padding: '2px 8px',
            borderRadius: 'var(--radius-sm)',
            border: '1px solid var(--line)',
            backgroundColor: 'var(--card)',
            color: 'var(--fg)',
            fontSize: 11,
            cursor: 'pointer',
          }}
        >
          {copied ? t('document.copied') : t('document.copyCode')}
        </button>
      </div>
      <pre
        style={{
          margin: 0,
          padding: '14px 16px',
          overflowX: 'auto',
          fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
          fontSize: 13,
          lineHeight: 1.6,
          color: 'var(--fg)',
        }}
      >
        <code>{code}</code>
      </pre>
    </div>
  )
}

function MarkdownBody({ markdown }: { markdown: string }) {
  const lines = markdown.replace(/\r\n/g, '\n').split('\n')
  const elements: React.ReactNode[] = []
  let i = 0
  let key = 0

  while (i < lines.length) {
    const line = lines[i]

    // Blank line
    if (line.trim() === '') {
      i++
      continue
    }

    // Fenced code block
    if (line.trim().startsWith('```')) {
      const lang = line.trim().slice(3).trim()
      const codeLines: string[] = []
      i++
      while (i < lines.length && !lines[i].trim().startsWith('```')) {
        codeLines.push(lines[i])
        i++
      }
      if (i < lines.length) i++ // skip closing ```
      elements.push(<CodeBlock key={`block-${key++}`} lang={lang} code={codeLines.join('\n')} />)
      continue
    }

    // Horizontal rule
    if (/^(-{3,}|\*{3,}|_{3,})$/.test(line.trim())) {
      elements.push(
        <hr
          key={`block-${key++}`}
          style={{ border: 'none', borderTop: '1px solid var(--line)', margin: '24px 0' }}
        />
      )
      i++
      continue
    }

    // Heading
    const headingMatch = line.match(/^(#{1,6})\s+(.*)$/)
    if (headingMatch) {
      const level = headingMatch[1].length
      const content = renderInlineMarkdown(headingMatch[2])
      const sizes: Record<number, number> = { 1: 24, 2: 20, 3: 17, 4: 15, 5: 14, 6: 13 }
      elements.push(
        <div
          key={`block-${key++}`}
          role="heading"
          aria-level={level}
          style={{
            margin: level <= 2 ? '24px 0 12px' : '18px 0 8px',
            paddingBottom: level <= 2 ? 6 : 0,
            borderBottom: level <= 2 ? '1px solid var(--line)' : 'none',
            fontSize: sizes[level] ?? 15,
            fontWeight: 700,
            color: 'var(--fg)',
            lineHeight: 1.4,
          }}
        >
          {content}
        </div>
      )
      i++
      continue
    }

    // Table
    if (
      line.includes('|') &&
      i + 1 < lines.length &&
      /^\s*\|?[\s:-]+\|[\s|:-]*$/.test(lines[i + 1])
    ) {
      const parseRow = (rowLine: string) =>
        rowLine
          .trim()
          .replace(/^\|/, '')
          .replace(/\|$/, '')
          .split('|')
          .map((cell) => cell.trim())

      const headers = parseRow(line)
      i += 2 // skip header and separator
      const rows: string[][] = []
      while (i < lines.length && lines[i].includes('|') && lines[i].trim() !== '') {
        rows.push(parseRow(lines[i]))
        i++
      }
      elements.push(
        <div key={`block-${key++}`} style={{ overflowX: 'auto', margin: '16px 0' }}>
          <table
            style={{
              width: '100%',
              borderCollapse: 'collapse',
              fontSize: 14,
              border: '1px solid var(--line)',
            }}
          >
            <thead>
              <tr style={{ backgroundColor: 'var(--bg)' }}>
                {headers.map((h, hIdx) => (
                  <th
                    key={hIdx}
                    style={{
                      padding: '8px 12px',
                      borderBottom: '1px solid var(--line)',
                      textAlign: 'left',
                      color: 'var(--fg)',
                      fontWeight: 600,
                    }}
                  >
                    {renderInlineMarkdown(h)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((r, rIdx) => (
                <tr key={rIdx} style={{ borderBottom: '1px solid var(--line)' }}>
                  {r.map((cell, cIdx) => (
                    <td
                      key={cIdx}
                      style={{
                        padding: '8px 12px',
                        color: 'var(--fg)',
                      }}
                    >
                      {renderInlineMarkdown(cell)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )
      continue
    }

    // Blockquote
    if (line.trim().startsWith('>')) {
      const quoteLines: string[] = []
      while (i < lines.length && lines[i].trim().startsWith('>')) {
        quoteLines.push(lines[i].trim().replace(/^>\s?/, ''))
        i++
      }
      elements.push(
        <blockquote
          key={`block-${key++}`}
          style={{
            margin: '16px 0',
            padding: '10px 16px',
            borderLeft: '3px solid var(--accent)',
            backgroundColor: 'var(--bg)',
            borderRadius: '0 var(--radius-sm) var(--radius-sm) 0',
            color: 'var(--muted)',
          }}
        >
          {renderInlineMarkdown(quoteLines.join(' '))}
        </blockquote>
      )
      continue
    }

    // Unordered list
    if (/^\s*[-*]\s+/.test(line)) {
      const items: string[] = []
      while (i < lines.length && /^\s*[-*]\s+/.test(lines[i])) {
        items.push(lines[i].replace(/^\s*[-*]\s+/, ''))
        i++
      }
      elements.push(
        <ul
          key={`block-${key++}`}
          style={{
            margin: '12px 0',
            paddingLeft: 22,
            color: 'var(--fg)',
            lineHeight: 1.75,
          }}
        >
          {items.map((item, idx) => (
            <li key={idx} style={{ marginBottom: 4 }}>
              {renderInlineMarkdown(item)}
            </li>
          ))}
        </ul>
      )
      continue
    }

    // Ordered list
    if (/^\s*\d+\.\s+/.test(line)) {
      const items: string[] = []
      while (i < lines.length && /^\s*\d+\.\s+/.test(lines[i])) {
        items.push(lines[i].replace(/^\s*\d+\.\s+/, ''))
        i++
      }
      elements.push(
        <ol
          key={`block-${key++}`}
          style={{
            margin: '12px 0',
            paddingLeft: 22,
            color: 'var(--fg)',
            lineHeight: 1.75,
          }}
        >
          {items.map((item, idx) => (
            <li key={idx} style={{ marginBottom: 4 }}>
              {renderInlineMarkdown(item)}
            </li>
          ))}
        </ol>
      )
      continue
    }

    // Paragraph
    const paraLines: string[] = []
    while (
      i < lines.length &&
      lines[i].trim() !== '' &&
      !lines[i].trim().startsWith('```') &&
      !lines[i].trim().startsWith('#') &&
      !lines[i].trim().startsWith('>') &&
      !/^\s*[-*]\s+/.test(lines[i]) &&
      !/^\s*\d+\.\s+/.test(lines[i])
    ) {
      paraLines.push(lines[i])
      i++
    }
    elements.push(
      <p
        key={`block-${key++}`}
        style={{
          margin: '12px 0',
          color: 'var(--fg)',
          fontSize: 15,
          lineHeight: 1.8,
        }}
      >
        {renderInlineMarkdown(paraLines.join(' '))}
      </p>
    )
  }

  return <div>{elements}</div>
}

export default function DocumentViewer({
  title,
  documentKey,
  documentMimeType,
  initialMarkdown,
}: Props) {
  const { t } = useLanguage()
  const viewerKind = resolveDocumentViewerKind(documentMimeType, documentKey)
  const resourceUrl =
    documentKey && !documentKey.includes('..')
      ? `/${documentKey.replace(/^\/+/, '')}`
      : null

  const [markdownText, setMarkdownText] = useState<string | null>(initialMarkdown ?? null)
  const [loading, setLoading] = useState<boolean>(
    viewerKind === 'markdown' && initialMarkdown == null && resourceUrl != null
  )
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (viewerKind !== 'markdown' || initialMarkdown != null || !resourceUrl) {
      return
    }
    let cancelled = false
    setLoading(true)
    setError(null)

    fetch(resourceUrl)
      .then((res) => {
        if (!res.ok) {
          throw new Error(`HTTP ${res.status}`)
        }
        return res.text()
      })
      .then((text) => {
        if (!cancelled) {
          setMarkdownText(text)
          setLoading(false)
        }
      })
      .catch(() => {
        if (!cancelled) {
          setError(t('document.loadError'))
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [viewerKind, resourceUrl, initialMarkdown, t])

  if (!resourceUrl && initialMarkdown == null) {
    return (
      <div
        style={{
          width: '100%',
          minHeight: 280,
          backgroundColor: 'var(--card)',
          border: '1px solid var(--line)',
          borderRadius: 'var(--radius-md)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: 'var(--muted)',
          fontSize: 14,
        }}
      >
        {t('document.notFound')}
      </div>
    )
  }

  return (
    <div
      style={{
        width: '100%',
        backgroundColor: 'var(--card)',
        border: '1px solid var(--line)',
        borderRadius: 'var(--radius-md)',
        overflow: 'hidden',
      }}
    >
      {/* Body */}
      {viewerKind === 'pdf' && resourceUrl ? (
        <iframe
          src={`${resourceUrl}#toolbar=0&navpanes=0`}
          title={title || 'PDF Document'}
          style={{
            width: '100%',
            height: '80vh',
            minHeight: 600,
            border: 'none',
            display: 'block',
            backgroundColor: '#ffffff',
          }}
        />
      ) : loading ? (
        <div
          style={{
            padding: '64px 24px',
            textAlign: 'center',
            color: 'var(--muted)',
            fontSize: 14,
          }}
        >
          {t('document.loading')}
        </div>
      ) : error ? (
        <div
          style={{
            padding: '48px 24px',
            textAlign: 'center',
            color: 'var(--warn)',
            fontSize: 14,
          }}
        >
          {error}
        </div>
      ) : (
        <div style={{ padding: '24px 28px' }}>
          <MarkdownBody markdown={markdownText ?? ''} />
        </div>
      )}
    </div>
  )
}
