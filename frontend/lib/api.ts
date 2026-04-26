const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "/api";

export type ContentType = "video" | "image_gallery" | "vr360" | "document";
export type ContentStatus = "pending" | "processing" | "ready" | "error";

export interface MongoContent {
  short_id: string;
  content_type: ContentType;
  title: string;
  description: string;
  duration_seconds: number | null;
  is_360: boolean;
  tags: string[];
  status: ContentStatus;
  disc_label: string | null;
  thumbnail_key: string | null;
  published_at: string | null;
  updated_at: string;
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
  tags: string[];
  status: ContentStatus;
}

export interface Content {
  id: string;
  short_id: string;
  content_type: ContentType;
  disc_id: string | null;
  title: string;
  description: string;
  duration_seconds: number | null;
  is_360: boolean;
  tags: string[];
  status: ContentStatus;
  published_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface Disc {
  id: string;
  label: string;
  disc_name: string | null;
  created_at: string;
}

export interface CreateContentRequest {
  content_type: ContentType;
  disc_id?: string | null;
  title: string;
  description?: string;
  duration_seconds?: number | null;
  is_360?: boolean;
  tags?: string[];
}

export interface UpdateContentRequest {
  title?: string | null;
  description?: string | null;
  tags?: string[] | null;
  status?: ContentStatus;
}

export interface CreateDiscRequest {
  label: string;
  disc_name?: string | null;
}

async function get<T>(path: string, params?: Record<string, string | number>): Promise<T> {
  const url = new URL(`${API_BASE}${path}`, typeof window !== "undefined" ? window.location.origin : "http://localhost");
  if (params) {
    Object.entries(params).forEach(([k, v]) => url.searchParams.set(k, String(v)));
  }
  const res = await fetch(url.toString(), { cache: "no-store" });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return res.json();
}

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return res.json();
}

async function put<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return res.json();
}

async function del(path: string): Promise<void> {
  const res = await fetch(`${API_BASE}${path}`, { method: "DELETE" });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
}

export const api = {
  contents: {
    list: (params?: { content_type?: ContentType; limit?: number; offset?: number }) =>
      get<MongoContent[]>("/v1/contents", params as Record<string, string | number>),
    get: (shortId: string) => get<MongoContent>(`/v1/contents/${shortId}`),
    variants: (shortId: string) => get<MongoVariant[]>(`/v1/contents/${shortId}/variants`),
  },
  search: (q: string, params?: { limit?: number; offset?: number }) =>
    get<SearchResult[]>("/v1/search", { q, ...params }),
  admin: {
    contents: {
      list: (params?: { status?: ContentStatus; limit?: number; offset?: number }) =>
        get<Content[]>("/v1/admin/contents", params as Record<string, string | number>),
      create: (body: CreateContentRequest) => post<Content>("/v1/admin/contents", body),
      update: (id: string, body: UpdateContentRequest) => put<Content>(`/v1/admin/contents/${id}`, body),
      delete: (id: string) => del(`/v1/admin/contents/${id}`),
    },
    discs: {
      list: () => get<Disc[]>("/v1/admin/discs"),
      create: (body: CreateDiscRequest) => post<Disc>("/v1/admin/discs", body),
    },
  },
};
