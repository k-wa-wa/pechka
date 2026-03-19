package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"pechka/streaming-service/api/internal/domain"
)

// CreateContentRequest is used to create any type of content.
type CreateContentRequest struct {
	ContentType  domain.ContentType    `json:"content_type"`
	Title        string                `json:"title"`
	Description  string                `json:"description"`
	Rating       *float64              `json:"rating,omitempty"`
	Tags         []string              `json:"tags,omitempty"`
	VideoDetails *domain.VideoDetails  `json:"video_details,omitempty"`
}

// AddAssetsRequest is used to attach asset files to a content.
type AddAssetsRequest struct {
	Assets []AssetRequest `json:"assets"`
}

type AssetRequest struct {
	Role      domain.AssetRole `json:"asset_role"`
	MinIOKey  string           `json:"minio_key"`
	PublicURL string           `json:"public_url,omitempty"`
}

// ContentUseCase defines the use case operations for content management.
type ContentUseCase interface {
	CreateContent(ctx context.Context, req CreateContentRequest) (*domain.Content, error)
	UpdateContent(ctx context.Context, id uuid.UUID, req CreateContentRequest) (*domain.Content, error)
	GetContentDetails(ctx context.Context, shortID string) (*domain.Content, error)
	ListContents(ctx context.Context) ([]*domain.Content, error)
	AddAssets(ctx context.Context, contentID uuid.UUID, req AddAssetsRequest) error
}

type contentUseCase struct {
	repo  domain.ContentRepository
	idGen domain.ShortIDGenerator
}

func NewContentUseCase(repo domain.ContentRepository, idGen domain.ShortIDGenerator) ContentUseCase {
	return &contentUseCase{
		repo:  repo,
		idGen: idGen,
	}
}

func (u *contentUseCase) CreateContent(ctx context.Context, req CreateContentRequest) (*domain.Content, error) {
	now := time.Now()
	c := &domain.Content{
		ID:           uuid.Must(uuid.NewV7()),
		ShortID:      u.idGen.Generate(),
		ContentType:  req.ContentType,
		Title:        req.Title,
		Description:  req.Description,
		Rating:       req.Rating,
		Tags:         req.Tags,
		PublishedAt:  &now,
		VideoDetails: req.VideoDetails,
	}
	if c.Tags == nil {
		c.Tags = []string{}
	}

	if err := u.repo.CreateContent(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *contentUseCase) UpdateContent(ctx context.Context, id uuid.UUID, req CreateContentRequest) (*domain.Content, error) {
	c := &domain.Content{
		ID:           id,
		Title:        req.Title,
		Description:  req.Description,
		Rating:       req.Rating,
		Tags:         req.Tags,
		VideoDetails: req.VideoDetails,
	}
	if c.Tags == nil {
		c.Tags = []string{}
	}
	if err := u.repo.UpdateContent(ctx, c); err != nil {
		return nil, err
	}
	return u.repo.GetContentByID(ctx, id)
}

func (u *contentUseCase) GetContentDetails(ctx context.Context, shortID string) (*domain.Content, error) {
	return u.repo.GetContentByShortID(ctx, shortID)
}

func (u *contentUseCase) ListContents(ctx context.Context) ([]*domain.Content, error) {
	return u.repo.ListContents(ctx)
}

func (u *contentUseCase) AddAssets(ctx context.Context, contentID uuid.UUID, req AddAssetsRequest) error {
	var assets []domain.Asset
	for _, a := range req.Assets {
		assets = append(assets, domain.Asset{
			ID:        uuid.Must(uuid.NewV7()),
			AssetRole: a.Role,
			S3Key:     a.MinIOKey,
			PublicURL: a.PublicURL,
		})
	}
	return u.repo.AddAssets(ctx, contentID, assets)
}
