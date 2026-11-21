-- 管理画面による字幕の手動編集（時刻のみの変更、キューの新規追加、キュー削除）を
-- subtitle_tracks 側で記録し、文字起こしバッチが本文差分のない手動編集トラックを上書き削除しないようにする。
-- 移行前から存在するトラックは時刻修正・行追加・行削除の有無を DB 上から判別できないため、
-- カラム追加時に既存トラックを一律保護し、移行後に生成バッチが新規作成する未編集 draft のみ DEFAULT FALSE で再生成可能とする。
BEGIN;

DO $$
DECLARE
    has_trigger BOOLEAN;
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'subtitle_tracks'
          AND column_name = 'manually_edited'
    ) THEN
        ALTER TABLE subtitle_tracks
            ADD COLUMN manually_edited BOOLEAN NOT NULL DEFAULT FALSE;

        IF EXISTS (SELECT 1 FROM subtitle_tracks) THEN
            SELECT EXISTS (
                SELECT 1
                FROM pg_trigger
                WHERE tgrelid = 'subtitle_tracks'::regclass
                  AND tgname = 'subtitle_tracks_updated_at'
                  AND tgenabled <> 'D'
            ) INTO has_trigger;

            IF has_trigger THEN
                EXECUTE 'ALTER TABLE subtitle_tracks DISABLE TRIGGER subtitle_tracks_updated_at';
            END IF;

            UPDATE subtitle_tracks
            SET manually_edited = TRUE;

            IF has_trigger THEN
                EXECUTE 'ALTER TABLE subtitle_tracks ENABLE TRIGGER subtitle_tracks_updated_at';
            END IF;
        END IF;
    END IF;
END;
$$;

COMMIT;
