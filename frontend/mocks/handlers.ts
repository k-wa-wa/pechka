import { http, HttpResponse } from 'msw'
import type { Content, SubtitleCue, SubtitleTrack, UpdateContentRequest } from '@/lib/types'
import {
  ADMIN_CONTENTS,
  CONTENTS,
  SEARCH_RESULTS,
  SUBTITLE_CUES,
  SUBTITLE_TRACKS,
  VARIANTS,
} from './fixtures'

// Mutable copies so stories can exercise create/update/delete flows without
// polluting the shared fixtures.
const adminContents: Content[] = ADMIN_CONTENTS.map((c) => ({ ...c }))
const subtitleTracks: Record<string, SubtitleTrack[]> = structuredClone(SUBTITLE_TRACKS)
const subtitleCues: Record<string, SubtitleCue[]> = structuredClone(SUBTITLE_CUES)

export const handlers = [
  http.get('/api/v1/contents', ({ request }) => {
    const url = new URL(request.url)
    const contentType = url.searchParams.get('content_type')
    const limit = Number(url.searchParams.get('limit') ?? '100')
    let items = CONTENTS
    if (contentType) items = items.filter((c) => c.content_type === contentType)
    return HttpResponse.json(items.slice(0, limit))
  }),

  http.get('/api/v1/contents/:shortId', ({ params }) => {
    const item = CONTENTS.find((c) => c.short_id === params.shortId)
    if (!item) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    return HttpResponse.json(item)
  }),

  http.get('/api/v1/contents/:shortId/variants', ({ params }) => {
    const variants = VARIANTS[params.shortId as string] ?? []
    return HttpResponse.json(variants)
  }),

  http.get('/api/v1/search', ({ request }) => {
    const url = new URL(request.url)
    const q = (url.searchParams.get('q') ?? '').toLowerCase()
    const results = SEARCH_RESULTS.filter(
      (c) =>
        c.title.toLowerCase().includes(q) ||
        c.description.toLowerCase().includes(q) ||
        c.tags.some((t) => t.toLowerCase().includes(q))
    )
    return HttpResponse.json(results)
  }),

  http.get('/api/v1/admin/contents', ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('status')
    const limit = Number(url.searchParams.get('limit') ?? '200')
    let items = adminContents
    if (status) items = items.filter((c) => c.status === status)
    return HttpResponse.json(items.slice(0, limit))
  }),

  http.put('/api/v1/admin/contents/:id', async ({ params, request }) => {
    const body = (await request.json()) as UpdateContentRequest
    const idx = adminContents.findIndex((c) => c.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    adminContents[idx] = {
      ...adminContents[idx],
      ...(body.title != null && { title: body.title }),
      ...(body.description != null && { description: body.description }),
      ...(body.tags != null && { tags: body.tags }),
      ...(body.status != null && { status: body.status }),
      updated_at: new Date().toISOString(),
    }
    return HttpResponse.json(adminContents[idx])
  }),

  http.post('/api/v1/admin/contents/:id/archive', ({ params }) => {
    const idx = adminContents.findIndex((c) => c.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    adminContents[idx] = { ...adminContents[idx], archived_at: new Date().toISOString() }
    return HttpResponse.json(adminContents[idx])
  }),

  http.post('/api/v1/admin/contents/:id/unarchive', ({ params }) => {
    const idx = adminContents.findIndex((c) => c.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    adminContents[idx] = { ...adminContents[idx], archived_at: null }
    return HttpResponse.json(adminContents[idx])
  }),

  http.get('/api/v1/admin/contents/:contentId/subtitles', ({ params }) => {
    const tracks = subtitleTracks[params.contentId as string] ?? []
    return HttpResponse.json(tracks)
  }),

  http.put('/api/v1/admin/subtitles/:trackId/status', async ({ params, request }) => {
    const body = (await request.json()) as { status: SubtitleTrack['status'] }
    for (const tracks of Object.values(subtitleTracks)) {
      const track = tracks.find((t) => t.id === params.trackId)
      if (track) {
        track.status = body.status
        track.updated_at = new Date().toISOString()
        return HttpResponse.json(track)
      }
    }
    return HttpResponse.json({ error: 'not found' }, { status: 404 })
  }),

  http.get('/api/v1/admin/subtitles/:trackId/cues', ({ params }) => {
    const cues = subtitleCues[params.trackId as string] ?? []
    return HttpResponse.json(cues)
  }),

  http.post('/api/v1/admin/subtitles/:trackId/cues', async ({ params, request }) => {
    const body = (await request.json()) as {
      seq: number
      start_ms: number
      end_ms: number
      text: string
    }
    const trackId = params.trackId as string
    const cue: SubtitleCue = {
      id: `cue-${Date.now()}`,
      track_id: trackId,
      seq: body.seq,
      start_ms: body.start_ms,
      end_ms: body.end_ms,
      text: body.text,
      original_text: body.text,
      flagged: false,
      updated_at: new Date().toISOString(),
    }
    subtitleCues[trackId] = [...(subtitleCues[trackId] ?? []), cue]
    return HttpResponse.json(cue)
  }),

  http.put('/api/v1/admin/subtitles/cues/:cueId', async ({ params, request }) => {
    const body = (await request.json()) as { text?: string; start_ms?: number; end_ms?: number }
    for (const cues of Object.values(subtitleCues)) {
      const cue = cues.find((c) => c.id === params.cueId)
      if (cue) {
        Object.assign(cue, body, { updated_at: new Date().toISOString() })
        return HttpResponse.json(cue)
      }
    }
    return HttpResponse.json({ error: 'not found' }, { status: 404 })
  }),

  http.delete('/api/v1/admin/subtitles/cues/:cueId', ({ params }) => {
    for (const [trackId, cues] of Object.entries(subtitleCues)) {
      const idx = cues.findIndex((c) => c.id === params.cueId)
      if (idx !== -1) {
        subtitleCues[trackId] = cues.filter((c) => c.id !== params.cueId)
        return new HttpResponse(null, { status: 204 })
      }
    }
    return HttpResponse.json({ error: 'not found' }, { status: 404 })
  }),
]
