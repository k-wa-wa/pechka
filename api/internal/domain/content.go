package domain

import "time"

type ContentType string
type ContentStatus string

const (
	ContentTypeVideo        ContentType = "video"
	ContentTypeImageGallery ContentType = "image_gallery"
	ContentTypeVR360        ContentType = "vr360"
	ContentTypeDocument     ContentType = "document"

	ContentStatusPending    ContentStatus = "pending"
	ContentStatusProcessing ContentStatus = "processing"
	ContentStatusReady      ContentStatus = "ready"
	ContentStatusError      ContentStatus = "error"
)

type Content struct {
	ID              string        `json:"id"`
	ShortID         string        `json:"short_id"`
	ContentType     ContentType   `json:"content_type"`
	DiscID          *string       `json:"disc_id,omitempty"`
	Title           string        `json:"title"`
	Description     string        `json:"description"`
	DurationSeconds *int          `json:"duration_seconds,omitempty"`
	Is360           bool          `json:"is_360"`
	Tags            []string      `json:"tags"`
	Status          ContentStatus `json:"status"`
	PublishedAt     *time.Time    `json:"published_at,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type VideoVariant struct {
	ID          string     `json:"id"`
	ContentID   string     `json:"content_id"`
	VariantType string     `json:"variant_type"`
	HLSKey      string     `json:"hls_key"`
	Bandwidth   *int       `json:"bandwidth,omitempty"`
	Resolution  *string    `json:"resolution,omitempty"`
	Codecs      *string    `json:"codecs,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Asset struct {
	ID        string    `json:"id"`
	ContentID string    `json:"content_id"`
	AssetRole string    `json:"asset_role"`
	S3Key     string    `json:"s3_key"`
	CreatedAt time.Time `json:"created_at"`
}

// MongoContent はMongoDB の非正規化ドキュメント構造
type MongoContent struct {
	ID              string         `bson:"_id" json:"short_id"`
	ContentType     ContentType    `bson:"content_type" json:"content_type"`
	Title           string         `bson:"title" json:"title"`
	Description     string         `bson:"description" json:"description"`
	DurationSeconds *int           `bson:"duration_seconds,omitempty" json:"duration_seconds,omitempty"`
	Is360           bool           `bson:"is_360" json:"is_360"`
	Tags            []string       `bson:"tags" json:"tags"`
	Status          ContentStatus  `bson:"status" json:"status"`
	DiscLabel       *string        `bson:"disc_label,omitempty" json:"disc_label,omitempty"`
	Variants        []MongoVariant `bson:"variants,omitempty" json:"variants,omitempty"`
	ThumbnailKey    *string        `bson:"thumbnail_key,omitempty" json:"thumbnail_key,omitempty"`
	PublishedAt     *time.Time     `bson:"published_at,omitempty" json:"published_at,omitempty"`
	UpdatedAt       time.Time      `bson:"updated_at" json:"updated_at"`
}

type MongoVariant struct {
	VariantType string  `bson:"variant_type" json:"variant_type"`
	HLSKey      string  `bson:"hls_key" json:"hls_key"`
	Bandwidth   *int    `bson:"bandwidth,omitempty" json:"bandwidth,omitempty"`
	Resolution  *string `bson:"resolution,omitempty" json:"resolution,omitempty"`
	Codecs      *string `bson:"codecs,omitempty" json:"codecs,omitempty"`
}
