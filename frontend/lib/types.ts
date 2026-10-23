export type ContentType = "video" | "image_gallery" | "vr360" | "document";
export type ContentStatus = "pending" | "processing" | "ready" | "error";
export type DocumentViewerKind = "markdown" | "pdf";

export interface DocumentFormatSpec {
  kind: DocumentViewerKind;
  mimeType: string;
  extensions: string[];
  label: string;
}

// 対応ドキュメント形式のレジストリ（新形式追加時はここにエントリを追加する）
export const DOCUMENT_FORMAT_SPECS: DocumentFormatSpec[] = [
  {
    kind: "markdown",
    mimeType: "text/markdown",
    extensions: [".md", ".markdown"],
    label: "Markdown",
  },
  {
    kind: "pdf",
    mimeType: "application/pdf",
    extensions: [".pdf"],
    label: "PDF",
  },
];

export const UPLOAD_ACCEPT_ATTR = [
  "video/*",
  ...DOCUMENT_FORMAT_SPECS.flatMap((s) => [...s.extensions, s.mimeType]),
].join(",");

export function resolveDocumentViewerKind(
  mimeType?: string | null,
  documentKey?: string | null,
): DocumentViewerKind | null {
  if (mimeType) {
    const normalized = mimeType.split(";")[0].trim().toLowerCase();
    const byMime = DOCUMENT_FORMAT_SPECS.find((s) => s.mimeType === normalized);
    if (byMime) return byMime.kind;
  }
  if (documentKey) {
    const lowerKey = documentKey.toLowerCase();
    const byExt = DOCUMENT_FORMAT_SPECS.find((s) =>
      s.extensions.some((ext) => lowerKey.endsWith(ext)),
    );
    if (byExt) return byExt.kind;
  }
  return null;
}

export interface Channel {
  id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface MongoContent {
  short_id: string;
  content_type: ContentType;
  channel_id?: string;
  channel_name?: string;
  title: string;
  description: string;
  duration_seconds: number | null;
  is_360: boolean;
  tags: string[];
  status: ContentStatus;
  disc_label: string | null;
  thumbnail_key: string | null;
  document_key?: string | null;
  document_mime_type?: string | null;
  published_at: string | null;
  updated_at: string;
  has_subtitles: boolean;
}

export interface MongoVariant {
  variant_type: string;
  hls_key: string;
  bandwidth: number | null;
  resolution: string | null;
  codecs: string | null;
}

export interface SearchResult {
  short_id: string;
  title: string;
  description: string;
  content_type: ContentType;
  channel_id?: string;
  channel_name?: string;
  tags: string[];
  status: ContentStatus;
}

export interface Content {
  id: string;
  short_id: string;
  content_type: ContentType;
  channel_id?: string;
  disc_id: string | null;
  title: string;
  description: string;
  duration_seconds: number | null;
  is_360: boolean;
  tags: string[];
  status: ContentStatus;
  document_key?: string | null;
  document_mime_type?: string | null;
  published_at: string | null;
  archived_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface AdminContentsResponse {
  contents: Content[];
  total: number;
  limit: number;
  offset: number;
}

export interface UpdateContentRequest {
  channel_id?: string | null;
  title?: string | null;
  description?: string | null;
  tags?: string[] | null;
  status?: ContentStatus;
}

export interface UploadContentRequest {
  title: string;
  description?: string;
  tags?: string[];
  channel_id?: string;
}

export interface UploadContentResponse {
  content: Content;
  workflow_name: string;
}

export type SubtitleTrackStatus = "draft" | "published";

export interface SubtitleTrack {
  id: string;
  content_id: string;
  language: string;
  status: SubtitleTrackStatus;
  model: string;
  created_at: string;
  updated_at: string;
}

export interface SubtitleCue {
  id: string;
  track_id: string;
  seq: number;
  start_ms: number;
  end_ms: number;
  text: string;
  original_text: string;
  flagged: boolean;
  updated_at: string;
}

export interface UpdateSubtitleCueRequest {
  text?: string;
  start_ms?: number;
  end_ms?: number;
}

export interface InsertSubtitleCueRequest {
  seq: number;
  start_ms: number;
  end_ms: number;
  text: string;
}
