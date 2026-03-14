package domain

import (
	"time"

	"github.com/google/uuid"
)

type ContentType string

const (
	ContentTypeVideo        ContentType = "video"
	ContentTypeVR360        ContentType = "vr360"
	ContentTypeImageGallery ContentType = "image_gallery"
	ContentTypeEbook        ContentType = "ebook"
)

type AssetRole string

const (
	AssetRoleThumbnail AssetRole = "thumbnail"
	AssetRoleHLSMaster AssetRole = "hls_master"
	AssetRolePDF       AssetRole = "pdf"
)

// Video represents the video domain entity
type Video struct {
	ID              uuid.UUID  `json:"id"`
	ShortID         string     `json:"short_id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Rating          *float64   `json:"rating,omitempty"`
	Is360           bool       `json:"is_360"`
	DurationSeconds int        `json:"duration_seconds"`
	Director        string     `json:"director,omitempty"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	Assets []Asset `json:"assets,omitempty"`
}

// Gallery represents the image gallery domain entity
type Gallery struct {
	ID          uuid.UUID  `json:"id"`
	ShortID     string     `json:"short_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Rating      *float64   `json:"rating,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	Assets []Asset `json:"assets,omitempty"`
}

// Asset represents physical files (images, videos, etc.) attached to a domain entity
type Asset struct {
	ID        uuid.UUID `json:"id"`
	AssetRole AssetRole `json:"asset_role"`
	S3Key     string    `json:"s3_key"`
	PublicURL string    `json:"public_url"`
}
