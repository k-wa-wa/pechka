import type {
  MongoContent,
  MongoVariant,
  SearchResult,
  Content,
  UpdateContentRequest,
  ContentStatus,
  ContentType,
  SubtitleTrack,
  SubtitleTrackStatus,
  SubtitleCue,
  UpdateSubtitleCueRequest,
  InsertSubtitleCueRequest,
} from './types'

// Server components use API_URL (internal k8s service); browser uses relative URL via nginx
const API_BASE =
  typeof window === 'undefined'
    ? (process.env.API_URL ?? 'http://api:8080')
    : (process.env.NEXT_PUBLIC_API_URL ?? '')

async function fetchJson<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, options)
  if (!res.ok) {
    throw new Error(`API error ${res.status}: ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getContents(params?: {
  limit?: number
  offset?: number
  content_type?: ContentType
}): Promise<MongoContent[]> {
  const query = new URLSearchParams()
  if (params?.limit != null) query.set('limit', String(params.limit))
  if (params?.offset != null) query.set('offset', String(params.offset))
  if (params?.content_type) query.set('content_type', params.content_type)
  const qs = query.toString()
  return fetchJson<MongoContent[]>(`${API_BASE}/api/v1/contents${qs ? `?${qs}` : ''}`, {
    cache: 'no-store',
  })
}

export async function getContent(shortId: string): Promise<MongoContent> {
  return fetchJson<MongoContent>(`${API_BASE}/api/v1/contents/${shortId}`, {
    cache: 'no-store',
  })
}

export async function getVariants(shortId: string): Promise<MongoVariant[]> {
  return fetchJson<MongoVariant[]>(`${API_BASE}/api/v1/contents/${shortId}/variants`, {
    cache: 'no-store',
  })
}

export async function searchContents(
  q: string,
  params?: { limit?: number; offset?: number }
): Promise<SearchResult[]> {
  const query = new URLSearchParams({ q })
  if (params?.limit != null) query.set('limit', String(params.limit))
  if (params?.offset != null) query.set('offset', String(params.offset))
  return fetchJson<SearchResult[]>(`${API_BASE}/api/v1/search?${query.toString()}`)
}

export async function getAdminContents(params?: {
  status?: ContentStatus
  limit?: number
  offset?: number
}): Promise<Content[]> {
  const query = new URLSearchParams()
  if (params?.status) query.set('status', params.status)
  if (params?.limit != null) query.set('limit', String(params.limit))
  if (params?.offset != null) query.set('offset', String(params.offset))
  const qs = query.toString()
  return fetchJson<Content[]>(`${API_BASE}/api/v1/admin/contents${qs ? `?${qs}` : ''}`, {
    cache: 'no-store',
  })
}

export async function updateContent(
  id: string,
  body: UpdateContentRequest
): Promise<Content> {
  return fetchJson<Content>(`${API_BASE}/api/v1/admin/contents/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

export async function getSubtitleTracks(contentId: string): Promise<SubtitleTrack[]> {
  return fetchJson<SubtitleTrack[]>(
    `${API_BASE}/api/v1/admin/contents/${contentId}/subtitles`,
    { cache: 'no-store' }
  )
}

export async function updateSubtitleTrackStatus(
  trackId: string,
  status: SubtitleTrackStatus
): Promise<SubtitleTrack> {
  return fetchJson<SubtitleTrack>(`${API_BASE}/api/v1/admin/subtitles/${trackId}/status`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status }),
  })
}

export async function getSubtitleCues(trackId: string): Promise<SubtitleCue[]> {
  return fetchJson<SubtitleCue[]>(`${API_BASE}/api/v1/admin/subtitles/${trackId}/cues`, {
    cache: 'no-store',
  })
}

export async function updateSubtitleCue(
  cueId: string,
  body: UpdateSubtitleCueRequest
): Promise<SubtitleCue> {
  return fetchJson<SubtitleCue>(`${API_BASE}/api/v1/admin/subtitles/cues/${cueId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

export async function insertSubtitleCue(
  trackId: string,
  body: InsertSubtitleCueRequest
): Promise<SubtitleCue> {
  return fetchJson<SubtitleCue>(`${API_BASE}/api/v1/admin/subtitles/${trackId}/cues`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

export async function deleteSubtitleCue(cueId: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/v1/admin/subtitles/cues/${cueId}`, {
    method: 'DELETE',
  })
  if (!res.ok) {
    throw new Error(`API error ${res.status}: ${res.statusText}`)
  }
}
