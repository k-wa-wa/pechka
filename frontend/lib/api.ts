import type {
  MongoContent,
  MongoVariant,
  SearchResult,
  Content,
  UpdateContentRequest,
  ContentStatus,
  ContentType,
} from './types'

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? ''

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
