export interface Group {
  id: string;
  name: string;
  description: string;
}

export interface Content {
  id: string;
  short_id: string;
  content_type: "video" | "image_gallery" | "vr360" | "ebook";
  title: string;
  description: string;
  rating: number | null;
  created_at: string;
  tags: string[];
  visibility: string;
  allowed_groups: string[];
  assets: Array<{ asset_role: string; public_url: string }>;
}
