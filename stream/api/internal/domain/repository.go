package domain

import (
	"context"

	"github.com/google/uuid"
)

type ContentRepository interface {
	// Video operations
	CreateVideo(ctx context.Context, video *Video) error
	GetVideoByShortID(ctx context.Context, shortID string) (*Video, error)
	GetVideoByID(ctx context.Context, id uuid.UUID) (*Video, error)
	UpdateVideo(ctx context.Context, video *Video) error
	ListVideos(ctx context.Context) ([]*Video, error)

	// Gallery operations
	CreateGallery(ctx context.Context, gallery *Gallery) error
	GetGalleryByShortID(ctx context.Context, shortID string) (*Gallery, error)
	GetGalleryByID(ctx context.Context, id uuid.UUID) (*Gallery, error)
	UpdateGallery(ctx context.Context, gallery *Gallery) error
	ListGalleries(ctx context.Context) ([]*Gallery, error)

	// Asset operations
	AddVideoAssets(ctx context.Context, videoID uuid.UUID, assets []Asset) error
	AddGalleryAssets(ctx context.Context, galleryID uuid.UUID, assets []Asset) error
	CheckDuplicateByS3Key(ctx context.Context, s3Key string) (bool, error) // Checks both tables
	ListAllShortIDs(ctx context.Context) ([]string, error)
}
