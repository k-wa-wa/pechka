package domain

import (
	"context"
	"time"
)

// CatalogContent represents the denormalized content document for the read-side (MongoDB)
type CatalogContent struct {
	ID            string                 `bson:"_id" json:"id"`
	ShortID       string                 `bson:"short_id" json:"short_id"`
	Type          ContentType            `bson:"type" json:"type"`
	Title         string                 `bson:"title" json:"title"`
	Description   string                 `bson:"description" json:"description"`
	Rating        float64                `bson:"rating" json:"rating"`
	Visibility    string                 `bson:"visibility" json:"visibility"`
	AllowedGroups []string               `bson:"allowed_groups" json:"allowed_groups"`
	Metadata      map[string]interface{} `bson:"metadata" json:"metadata"`
	Assets        map[string]string      `bson:"assets" json:"assets"` // Role -> S3Key/URL
	Tags          []string               `bson:"tags" json:"tags"`
	UpdatedAt     time.Time              `bson:"updated_at" json:"updated_at"`
}

// CatalogRepository defines operations for the read-optimized document store
type CatalogRepository interface {
	Upsert(ctx context.Context, content *CatalogContent) error
	GetByShortID(ctx context.Context, shortID string, userGroups []string) (*CatalogContent, error)
	Search(ctx context.Context, query string, userGroups []string) ([]*CatalogContent, error)
	GetByIDs(ctx context.Context, ids []string, userGroups []string) ([]*CatalogContent, error)
}

// SearchRepository defines operations for the search index (Elasticsearch)
type SearchRepository interface {
	SearchIDs(ctx context.Context, query string, tags []string, userGroups []string) ([]string, error)
}
