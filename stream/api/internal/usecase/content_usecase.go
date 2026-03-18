package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"pechka/streaming-service/api/internal/domain"
)

type CreateVideoRequest struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Rating          *float64 `json:"rating,omitempty"`
	Is360           bool     `json:"is_360"`
	DurationSeconds int      `json:"duration_seconds"`
	Director        string   `json:"director,omitempty"`
}

type CreateGalleryRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Rating      *float64 `json:"rating,omitempty"`
}

type AddAssetsRequest struct {
	Assets []AssetRequest `json:"assets"`
}

type AssetRequest struct {
	Role     domain.AssetRole `json:"asset_role"`
	MinIOKey string           `json:"minio_key"`
}

type ContentUseCase interface {
	CreateVideo(ctx context.Context, req CreateVideoRequest) (*domain.Video, error)
	CreateGallery(ctx context.Context, req CreateGalleryRequest) (*domain.Gallery, error)
	AddVideoAssets(ctx context.Context, videoID uuid.UUID, req AddAssetsRequest) error
	AddGalleryAssets(ctx context.Context, galleryID uuid.UUID, req AddAssetsRequest) error
	GetVideoDetails(ctx context.Context, shortID string) (*domain.Video, error)
	GetGalleryDetails(ctx context.Context, shortID string) (*domain.Gallery, error)
	ListVideos(ctx context.Context) ([]*domain.Video, error)
	ListGalleries(ctx context.Context) ([]*domain.Gallery, error)
	UpdateVideo(ctx context.Context, id uuid.UUID, v *domain.Video) error
	UpdateGallery(ctx context.Context, id uuid.UUID, g *domain.Gallery) error
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

func (u *contentUseCase) CreateVideo(ctx context.Context, req CreateVideoRequest) (*domain.Video, error) {
	videoID := uuid.Must(uuid.NewV7())
	shortID := u.idGen.Generate()

	v := &domain.Video{
		ID:              videoID,
		ShortID:         shortID,
		Title:           req.Title,
		Description:     req.Description,
		Rating:          req.Rating,
		Is360:           req.Is360,
		DurationSeconds: req.DurationSeconds,
		Director:        req.Director,
		PublishedAt:     func() *time.Time { t := time.Now(); return &t }(),
	}

	if err := u.repo.CreateVideo(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (u *contentUseCase) CreateGallery(ctx context.Context, req CreateGalleryRequest) (*domain.Gallery, error) {
	galleryID := uuid.Must(uuid.NewV7())
	shortID := u.idGen.Generate()

	g := &domain.Gallery{
		ID:          galleryID,
		ShortID:     shortID,
		Title:       req.Title,
		Description: req.Description,
		Rating:      req.Rating,
		PublishedAt: func() *time.Time { t := time.Now(); return &t }(),
	}

	if err := u.repo.CreateGallery(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (u *contentUseCase) AddVideoAssets(ctx context.Context, id uuid.UUID, req AddAssetsRequest) error {
	var assets []domain.Asset
	for _, a := range req.Assets {
		assets = append(assets, domain.Asset{
			ID:        uuid.Must(uuid.NewV7()),
			AssetRole: a.Role,
			S3Key:     a.MinIOKey,
		})
	}

	return u.repo.AddVideoAssets(ctx, id, assets)
}

func (u *contentUseCase) AddGalleryAssets(ctx context.Context, id uuid.UUID, req AddAssetsRequest) error {
	var assets []domain.Asset
	for _, a := range req.Assets {
		assets = append(assets, domain.Asset{
			ID:        uuid.Must(uuid.NewV7()),
			AssetRole: a.Role,
			S3Key:     a.MinIOKey,
		})
	}

	return u.repo.AddGalleryAssets(ctx, id, assets)
}

func (u *contentUseCase) GetVideoDetails(ctx context.Context, shortID string) (*domain.Video, error) {
	return u.repo.GetVideoByShortID(ctx, shortID)
}

func (u *contentUseCase) GetGalleryDetails(ctx context.Context, shortID string) (*domain.Gallery, error) {
	return u.repo.GetGalleryByShortID(ctx, shortID)
}

func (u *contentUseCase) ListVideos(ctx context.Context) ([]*domain.Video, error) {
	return u.repo.ListVideos(ctx)
}

func (u *contentUseCase) ListGalleries(ctx context.Context) ([]*domain.Gallery, error) {
	return u.repo.ListGalleries(ctx)
}

func (u *contentUseCase) UpdateVideo(ctx context.Context, id uuid.UUID, v *domain.Video) error {
	v.ID = id
	if err := u.repo.UpdateVideo(ctx, v); err != nil {
		return err
	}
	return nil
}

func (u *contentUseCase) UpdateGallery(ctx context.Context, id uuid.UUID, g *domain.Gallery) error {
	g.ID = id
	if err := u.repo.UpdateGallery(ctx, g); err != nil {
		return err
	}
	return nil
}

