BEGIN;

ALTER TABLE subtitle_tracks DROP COLUMN IF EXISTS manually_edited;

COMMIT;
