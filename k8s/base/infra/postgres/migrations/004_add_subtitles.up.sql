-- 字幕生成パイプライン用のテーブルを追加する。
-- cue（行）単位でPostgresに保持し、admin画面から個別に編集できるようにする。
-- 公開判定は subtitle_tracks.status で行い、published のトラックのみ
-- Benthos CDC経由でMongoDBに同期されて視聴者に配信される。

CREATE TABLE IF NOT EXISTS subtitle_tracks (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    content_id UUID         NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    language   VARCHAR(10)  NOT NULL DEFAULT 'ja',
    status     VARCHAR(20)  NOT NULL DEFAULT 'draft', -- draft | published
    model      VARCHAR(50)  NOT NULL DEFAULT 'large-v3',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_subtitle_tracks_content_lang
    ON subtitle_tracks(content_id, language);

CREATE TABLE IF NOT EXISTS subtitle_cues (
    id            UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    track_id      UUID    NOT NULL REFERENCES subtitle_tracks(id) ON DELETE CASCADE,
    seq           INTEGER NOT NULL,
    start_ms      INTEGER NOT NULL,
    end_ms        INTEGER NOT NULL,
    text          TEXT    NOT NULL,       -- 編集可能な表示テキスト
    original_text TEXT    NOT NULL,       -- Whisperの生出力（diff/revert用に保持）
    flagged       BOOLEAN NOT NULL DEFAULT FALSE, -- ハルシネーション疑いのフラグ
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 挿入時に後続cueのseqを+1シフトする処理(1トランザクション内でUPDATE→INSERT)が
-- 一時的にseqを重複させうるため、コミット時にまとめて検査するDEFERRABLE制約にする。
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_subtitle_cues_track_seq'
    ) THEN
        ALTER TABLE subtitle_cues
            ADD CONSTRAINT uq_subtitle_cues_track_seq
            UNIQUE (track_id, seq) DEFERRABLE INITIALLY DEFERRED;
    END IF;
END;
$$;

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS subtitle_tracks_updated_at ON subtitle_tracks;
CREATE TRIGGER subtitle_tracks_updated_at
    BEFORE UPDATE ON subtitle_tracks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

DROP TRIGGER IF EXISTS subtitle_cues_updated_at ON subtitle_cues;
CREATE TRIGGER subtitle_cues_updated_at
    BEFORE UPDATE ON subtitle_cues
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
