export interface Group {
  id: string;
  name: string;
  description: string;
}

export interface GroupPermission {
  group_id: string;
  can_read: boolean;
  can_write: boolean;
  can_delete: boolean;
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
  group_permissions: GroupPermission[];
  assets: Array<{ asset_role: string; public_url: string }>;
}
