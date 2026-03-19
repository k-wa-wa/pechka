package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ContentType represents the type of content
type ContentType string

const (
	ContentTypeVideo        ContentType = "video"
	ContentTypeVR360        ContentType = "vr360"
	ContentTypeImageGallery ContentType = "image_gallery"
	ContentTypeEbook        ContentType = "ebook"
)

// AssetRole represents the role of a file asset
type AssetRole string

const (
	AssetRoleThumbnail AssetRole = "thumbnail"
	AssetRoleHLSMaster AssetRole = "hls_master"
	AssetRolePDF       AssetRole = "pdf"
)

// VideoDetails holds fields specific to video content (maps to content_videos table)
type VideoDetails struct {
	Is360           bool   `json:"is_360"`
	DurationSeconds int    `json:"duration_seconds"`
	Director        string `json:"director,omitempty"`
}

// Content represents the unified content domain entity (maps to contents table)
type Content struct {
	ID          uuid.UUID    `json:"id"`
	ShortID     string       `json:"short_id"`
	ContentType ContentType  `json:"content_type"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Rating      *float64     `json:"rating,omitempty"`
	Tags        []string     `json:"tags"`
	PublishedAt *time.Time   `json:"published_at,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`

	// VideoDetails is non-nil only for ContentTypeVideo and ContentTypeVR360
	VideoDetails *VideoDetails `json:"video_details,omitempty"`

	Assets []Asset `json:"assets,omitempty"`
}

// Asset represents a physical file attached to a content (maps to assets table)
type Asset struct {
	ID        uuid.UUID `json:"id"`
	AssetRole AssetRole `json:"asset_role"`
	S3Key     string    `json:"s3_key"`
	PublicURL string    `json:"public_url"`
}

// ScorerResult is returned by the ThumbnailScorer
type ScorerResult struct {
	BestTimestamp float64 `json:"best_timestamp"`
}

// ThumbnailScorer analyzes video frames to determine the best thumbnail timestamp
type ThumbnailScorer interface {
	Analyze(ctx context.Context, path string, points []float64) (*ScorerResult, error)
}
