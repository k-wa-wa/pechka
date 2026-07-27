'use client'

import { useEffect, useState } from 'react'
import type { Content, SubtitleCue, SubtitleTrack } from '@/lib/types'
import {
  deleteSubtitleCue,
  getSubtitleCues,
  getSubtitleTracks,
  insertSubtitleCue,
  updateSubtitleCue,
  updateSubtitleTrackStatus,
} from '@/lib/api'

interface Props {
  content: Content
  onClose: () => void
}

function formatMs(ms: number): string {
  const totalSec = ms / 1000
  const m = Math.floor(totalSec / 60)
  const s = (totalSec % 60).toFixed(3).padStart(6, '0')
  return `${m}:${s}`
}

export default function SubtitleEditorModal({ content, onClose }: Props) {
  const [tracks, setTracks] = useState<SubtitleTrack[]>([])
  const [selectedTrack, setSelectedTrack] = useState<SubtitleTrack | null>(null)
  const [cues, setCues] = useState<SubtitleCue[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [savingCueId, setSavingCueId] = useState<string | null>(null)

  useEffect(() => {
    getSubtitleTracks(content.id)
      .then((t) => {
        setTracks(t)
        if (t.length > 0) setSelectedTrack(t[0])
      })
      .catch((e) => setError(e instanceof Error ? e.message : String(e)))
      .finally(() => setLoading(false))
  }, [content.id])

  useEffect(() => {
    if (!selectedTrack) {
      setCues([])
      return
    }
    getSubtitleCues(selectedTrack.id)
      .then(setCues)
      .catch((e) => setError(e instanceof Error ? e.message : String(e)))
  }, [selectedTrack])

  async function handleTextChange(cue: SubtitleCue, text: string) {
    setSavingCueId(cue.id)
    try {
      const updated = await updateSubtitleCue(cue.id, { text })
      setCues((prev) => prev.map((c) => (c.id === cue.id ? updated : c)))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSavingCueId(null)
    }
  }

  async function handleDelete(cue: SubtitleCue) {
    try {
      await deleteSubtitleCue(cue.id)
      setCues((prev) => prev.filter((c) => c.id !== cue.id))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  async function handleInsertAfter(cue: SubtitleCue) {
    if (!selectedTrack) return
    try {
      const created = await insertSubtitleCue(selectedTrack.id, {
        seq: cue.seq + 1,
        start_ms: cue.end_ms,
        end_ms: cue.end_ms + 2000,
        text: '',
      })
      setCues((prev) => {
        const idx = prev.findIndex((c) => c.id === cue.id)
        const next = [...prev]
        next.splice(idx + 1, 0, created)
        return next
      })
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  async function handleTogglePublish() {
    if (!selectedTrack) return
    const nextStatus = selectedTrack.status === 'published' ? 'draft' : 'published'
    try {
      const updated = await updateSubtitleTrackStatus(selectedTrack.id, nextStatus)
      setSelectedTrack(updated)
      setTracks((prev) => prev.map((t) => (t.id === updated.id ? updated : t)))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  const flaggedCount = cues.filter((c) => c.flagged).length

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 200,
        backgroundColor: 'rgba(0,0,0,0.7)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 16,
      }}
      onClick={onClose}
    >
      <div
        style={{
          width: '100%',
          maxWidth: 820,
          maxHeight: '85vh',
          display: 'flex',
          flexDirection: 'column',
          backgroundColor: '#161b22',
          border: '1px solid #30363d',
          borderRadius: 12,
          overflow: 'hidden',
          boxShadow: '0 24px 48px rgba(0,0,0,0.5)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '16px 20px',
            borderBottom: '1px solid #30363d',
          }}
        >
          <h2 style={{ margin: 0, fontSize: 16, color: '#e6edf3' }}>
            字幕編集 — {content.title}
          </h2>
          <button
            onClick={onClose}
            style={{ background: 'none', border: 'none', color: '#8b949e', cursor: 'pointer', padding: 4 }}
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div style={{ flex: 1, overflowY: 'auto', padding: '16px 20px' }}>
          {loading && <p style={{ color: '#8b949e' }}>読み込み中...</p>}

          {!loading && tracks.length === 0 && (
            <p style={{ color: '#8b949e' }}>
              このコンテンツには字幕がありません
            </p>
          )}

          {!loading && tracks.length > 0 && selectedTrack && (
            <>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  marginBottom: 12,
                  gap: 12,
                  flexWrap: 'wrap',
                }}
              >
                <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                  <span
                    style={{
                      fontSize: 12,
                      fontWeight: 600,
                      padding: '2px 8px',
                      borderRadius: 4,
                      backgroundColor: selectedTrack.status === 'published' ? '#23863622' : '#8b949e22',
                      color: selectedTrack.status === 'published' ? '#3fb950' : '#8b949e',
                    }}
                  >
                    {selectedTrack.status === 'published' ? '公開中' : '下書き'}
                  </span>
                  <span style={{ fontSize: 12, color: '#8b949e' }}>
                    {selectedTrack.language} / {selectedTrack.model} / {cues.length}行
                    {flaggedCount > 0 && (
                      <span style={{ color: '#d29922' }}> ・要確認 {flaggedCount}件</span>
                    )}
                  </span>
                </div>
                <button
                  onClick={handleTogglePublish}
                  style={{
                    padding: '6px 14px',
                    borderRadius: 6,
                    border: 'none',
                    backgroundColor: selectedTrack.status === 'published' ? '#30363d' : '#238636',
                    color: '#e6edf3',
                    cursor: 'pointer',
                    fontSize: 13,
                    fontWeight: 600,
                  }}
                >
                  {selectedTrack.status === 'published' ? '非公開に戻す' : '公開する'}
                </button>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                {cues.map((cue) => (
                  <div
                    key={cue.id}
                    style={{
                      display: 'flex',
                      gap: 10,
                      alignItems: 'flex-start',
                      padding: 10,
                      borderRadius: 6,
                      backgroundColor: cue.flagged ? '#d2992218' : '#0d1117',
                      border: cue.flagged ? '1px solid #d2992255' : '1px solid #21262d',
                    }}
                  >
                    <span
                      style={{
                        fontSize: 11,
                        color: '#8b949e',
                        whiteSpace: 'nowrap',
                        paddingTop: 6,
                        minWidth: 96,
                      }}
                      title={cue.text !== cue.original_text ? `元: ${cue.original_text}` : undefined}
                    >
                      {formatMs(cue.start_ms)} → {formatMs(cue.end_ms)}
                    </span>
                    <textarea
                      defaultValue={cue.text}
                      rows={1}
                      onBlur={(e) => {
                        if (e.target.value !== cue.text) handleTextChange(cue, e.target.value)
                      }}
                      style={{
                        flex: 1,
                        backgroundColor: 'transparent',
                        border: 'none',
                        color: '#e6edf3',
                        fontSize: 14,
                        outline: 'none',
                        resize: 'vertical',
                        fontFamily: 'inherit',
                        opacity: savingCueId === cue.id ? 0.5 : 1,
                      }}
                    />
                    <div style={{ display: 'flex', gap: 4 }}>
                      <button
                        onClick={() => handleInsertAfter(cue)}
                        title="この行の後に挿入"
                        style={{
                          background: 'none',
                          border: '1px solid #30363d',
                          borderRadius: 4,
                          color: '#8b949e',
                          cursor: 'pointer',
                          fontSize: 12,
                          padding: '2px 6px',
                        }}
                      >
                        +
                      </button>
                      <button
                        onClick={() => handleDelete(cue)}
                        title="削除"
                        style={{
                          background: 'none',
                          border: '1px solid #30363d',
                          borderRadius: 4,
                          color: '#ff7b72',
                          cursor: 'pointer',
                          fontSize: 12,
                          padding: '2px 6px',
                        }}
                      >
                        ×
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </>
          )}

          {error && (
            <div
              style={{
                marginTop: 12,
                backgroundColor: '#ff7b7222',
                border: '1px solid #ff7b7244',
                borderRadius: 6,
                padding: '8px 12px',
                color: '#ff7b72',
                fontSize: 13,
              }}
            >
              {error}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
