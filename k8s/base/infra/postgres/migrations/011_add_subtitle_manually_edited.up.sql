-- 管理画面による字幕の手動編集（時刻のみの変更、キューの新規追加、キュー削除）を
-- subtitle_tracks 側で記録し、文字起こしバッチが本文差分のない手動編集トラックを上書き削除しないようにする。
ALTER TABLE subtitle_tracks ADD COLUMN IF NOT EXISTS manually_edited BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE subtitle_tracks st
SET manually_edited = TRUE
WHERE manually_edited = FALSE
  AND EXISTS (
      SELECT 1
      FROM subtitle_cues cu
      WHERE cu.track_id = st.id
        AND cu.text <> cu.original_text
  );
