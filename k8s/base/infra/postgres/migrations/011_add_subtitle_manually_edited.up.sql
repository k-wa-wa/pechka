-- 管理画面による字幕の手動編集（時刻のみの変更、キューの新規追加、キュー削除）を
-- subtitle_tracks 側で記録し、文字起こしバッチが本文差分のない手動編集トラックを上書き削除しないようにする。
-- 移行前から存在するトラックは時刻修正・行追加・行削除の有無を DB 上から判別できないため、
-- 初回補完時に既存トラックを一律保護し、移行後に生成バッチが新規作成する未編集 draft のみ DEFAULT FALSE で再生成可能とする。
-- 旧版 011（本文差分のみを TRUE にしていた版）から更新する環境でも一度だけ全既存トラックへ補完を適用し、
-- 以降のマイグレーション再実行では新規の未編集 draft が保たれるよう、列コメントに完了マーカーを記録する。
BEGIN;

DO $$
DECLARE
    has_trigger BOOLEAN;
    backfill_marker TEXT;
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
    END IF;

    SELECT col_description(c.oid, a.attnum)
    INTO backfill_marker
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN pg_attribute a ON a.attrelid = c.oid
    WHERE n.nspname = current_schema()
      AND c.relname = 'subtitle_tracks'
      AND a.attname = 'manually_edited'
      AND NOT a.attisdropped;

    IF backfill_marker IS DISTINCT FROM 'pechka:011:backfilled_all_existing' THEN
        IF EXISTS (SELECT 1 FROM subtitle_tracks WHERE manually_edited = FALSE) THEN
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
            SET manually_edited = TRUE
            WHERE manually_edited = FALSE;

            IF has_trigger THEN
                EXECUTE 'ALTER TABLE subtitle_tracks ENABLE TRIGGER subtitle_tracks_updated_at';
            END IF;
        END IF;

        EXECUTE 'COMMENT ON COLUMN subtitle_tracks.manually_edited IS ''pechka:011:backfilled_all_existing''';
    END IF;
END;
$$;

COMMIT;
