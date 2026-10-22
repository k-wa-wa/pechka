-- チャンネル（コンテンツ提供者）のマスターテーブルを作成し、
-- すべてのコンテンツをチャンネルに紐付ける。

CREATE TABLE IF NOT EXISTS channels (
    id          VARCHAR(50)  PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

DROP TRIGGER IF EXISTS channels_updated_at ON channels;
CREATE TRIGGER channels_updated_at
    BEFORE UPDATE ON channels
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- マスターデータの初期投入（冪等）
INSERT INTO channels (id, name, description) VALUES
    ('tech-feed', 'Tech Feed', '自動収集・AI生成による技術ダイジェスト解説'),
    ('bluray', 'Blu-ray', 'Blu-ray取り込み動画')
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description;

-- contents テーブルに channel_id カラムを追加（既存行の初期マッピングのため一旦NULL許容で追加）
ALTER TABLE contents ADD COLUMN IF NOT EXISTS channel_id VARCHAR(50) REFERENCES channels(id);

-- 既存コンテンツの自動マッピング（未設定の行のみ対象とすることで再実行時の上書きを防ぐ）
UPDATE contents
SET channel_id = CASE
    WHEN 'tech-feed' = ANY(tags) OR short_id LIKE 'tech-feed-%' THEN 'tech-feed'
    ELSE 'bluray'
END
WHERE channel_id IS NULL;

-- マッピング完了後に NOT NULL 制約を設定（デフォルト値は持たせず明示指定を必須とする）
ALTER TABLE contents ALTER COLUMN channel_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_contents_channel_id ON contents(channel_id);
