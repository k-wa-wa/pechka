/**
 * サムネイル URL 正規化およびプレースホルダーユーティリティ
 */

/**
 * thumbnail_key から正規化されたサムネイル画像 URL を生成する。
 * 先頭の "thumbnails/" (または "/thumbnails/") の有無に関わらず、
 * 常に "/thumbnails/<path>" 形式の URL を返す。
 */
export function getThumbnailUrl(thumbnailKey?: string | null): string | null {
  if (!thumbnailKey) {
    return null;
  }
  const cleanKey = thumbnailKey.replace(/^(\/?thumbnails\/)+/, "");
  if (!cleanKey) {
    return null;
  }
  return `/thumbnails/${cleanKey}`;
}

/**
 * content_type に応じたプレースホルダー画像の URL を返す。
 */
export function getPlaceholderUrl(contentType?: string | null): string {
  const type = contentType || "video";
  return `/images/placeholder-${type}.svg`;
}
