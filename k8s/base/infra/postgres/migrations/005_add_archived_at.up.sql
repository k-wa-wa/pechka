-- コンテンツのアーカイブ(論理削除)機能のため、アーカイブ日時を保持する列を追加する
-- NULL の場合は未アーカイブ、値が入っていればアーカイブ済みを表す
-- 削除は行わず、admin画面以外(公開API/検索)から見えなくするためのフラグとして利用する
ALTER TABLE contents ADD COLUMN IF NOT EXISTS archived_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_contents_archived_at ON contents(archived_at);
