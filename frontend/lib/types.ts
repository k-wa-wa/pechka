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
  created_at: string
  updated_at: string
}

export interface UpdateContentRequest {
  title?: string | null
  description?: string | null
  tags?: string[] | null
  status?: ContentStatus
}
