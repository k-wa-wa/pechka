UPDATE contents
SET thumbnail_key = regexp_replace(thumbnail_key, '^thumbnails/', '')
WHERE thumbnail_key LIKE 'thumbnails/%';
