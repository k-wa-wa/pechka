-- 既存の contents.thumbnail_key で "thumbnails/" が欠けているレコードを補正する
-- Benthos CDC 経由で MongoDB / Elasticsearch にも自動反映される
UPDATE contents
SET thumbnail_key = 'thumbnails/' || thumbnail_key
WHERE thumbnail_key IS NOT NULL
  AND thumbnail_key NOT LIKE 'thumbnails/%';
