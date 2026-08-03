export type ContentType = 'video' | 'image_gallery' | 'vr360' | 'document'
export type ContentStatus = 'pending' | 'processing' | 'ready' | 'error'

export interface MongoContent {
  short_id: string
  content_type: ContentType
  title: string
  description: string
  duration_seconds: number | null
  is_360: boolean
  tags: string[]
  status: ContentStatus
  disc_label: string | null
  thumbnail_key: string | null
  published_at: string | null
  updated_at: string
  has_subtitles: boolean
}

export interface MongoVariant {
  variant_type: string
  hls_key: string
  bandwidth: number | null
  resolution: string | null
  codecs: string | null
}

export interface SearchResult {
  short_id: string
  title: string
  description: string
  content_type: ContentType
  tags: string[]
  status: ContentStatus
}

export interface Content {
  id: string
  short_id: string
  content_type: ContentType
  disc_id: string | null
  title: string
  description: string
  duration_seconds: number | null
  is_360: boolean
  tags: string[]
  status: ContentStatus
  published_at: string | null
  archived_at: string | null
  created_at: string
  updated_at: string
}

export interface UpdateContentRequest {
  title?: string | null
  description?: string | null
  tags?: string[] | null
  status?: ContentStatus
}

export type SubtitleTrackStatus = 'draft' | 'published'

export interface SubtitleTrack {
  id: string
  content_id: string
  language: string
  status: SubtitleTrackStatus
  model: string
  created_at: string
  updated_at: string
}

export interface SubtitleCue {
  id: string
  track_id: string
  seq: number
  start_ms: number
  end_ms: number
  text: string
  original_text: string
  flagged: boolean
  updated_at: string
}

export interface UpdateSubtitleCueRequest {
  text?: string
  start_ms?: number
  end_ms?: number
}

export interface InsertSubtitleCueRequest {
  seq: number
  start_ms: number
  end_ms: number
  text: string
}
