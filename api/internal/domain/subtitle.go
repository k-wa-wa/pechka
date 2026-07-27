package domain

import "time"

type SubtitleTrackStatus string

const (
	SubtitleTrackStatusDraft     SubtitleTrackStatus = "draft"
	SubtitleTrackStatusPublished SubtitleTrackStatus = "published"
)

// SubtitleTrack は admin 編集画面向けの Postgres 上の字幕トラック（書き込み系）
type SubtitleTrack struct {
	ID        string              `json:"id"`
	ContentID string              `json:"content_id"`
	Language  string              `json:"language"`
	Status    SubtitleTrackStatus `json:"status"`
	Model     string              `json:"model"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

// SubtitleCue は admin 編集画面向けの Postgres 上の字幕1行（書き込み系）
type SubtitleCue struct {
	ID           string    `json:"id"`
	TrackID      string    `json:"track_id"`
	Seq          int       `json:"seq"`
	StartMs      int       `json:"start_ms"`
	EndMs        int       `json:"end_ms"`
	Text         string    `json:"text"`
	OriginalText string    `json:"original_text"`
	Flagged      bool      `json:"flagged"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// MongoSubtitle は published トラックのみが同期される配信用ドキュメント（読み取り系）
type MongoSubtitle struct {
	ID       string     `bson:"_id" json:"-"`
	ShortID  string     `bson:"short_id" json:"short_id"`
	Language string     `bson:"language" json:"language"`
	Status   string     `bson:"status" json:"status"`
	Cues     []MongoCue `bson:"cues" json:"cues"`
}

type MongoCue struct {
	StartMs int    `bson:"start_ms" json:"start_ms"`
	EndMs   int    `bson:"end_ms" json:"end_ms"`
	Text    string `bson:"text" json:"text"`
}
