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
	ParentType string         `json:"parent_type"` // "video" or "gallery"
	Assets     []AssetRequest `json:"assets"`
}

type AssetRequest struct {
	Role     domain.AssetRole `json:"asset_role"`
	MinIOKey string           `json:"minio_key"`
}

type ContentUseCase interface {
	CreateVideo(ctx context.Context, req CreateVideoRequest) (*domain.Video, error)
	CreateGallery(ctx context.Context, req CreateGalleryRequest) (*domain.Gallery, error)
	AddAssets(ctx context.Context, parentID uuid.UUID, req AddAssetsRequest) error
	GetVideoDetails(ctx context.Context, shortID string) (*domain.Video, error)
	GetGalleryDetails(ctx context.Context, shortID string) (*domain.Gallery, error)
	ListVideos(ctx context.Context) ([]*domain.Video, error)
	ListGalleries(ctx context.Context) ([]*domain.Gallery, error)
	UpdateVideo(ctx context.Context, id uuid.UUID, v *domain.Video) error
	UpdateGallery(ctx context.Context, id uuid.UUID, g *domain.Gallery) error
}

type contentUseCase struct {
	repo           domain.ContentRepository
	catalogSyncURL string
	idGen          domain.ShortIDGenerator
}

func NewContentUseCase(repo domain.ContentRepository, catalogSyncURL string, idGen domain.ShortIDGenerator) ContentUseCase {
	return &contentUseCase{
		repo:           repo,
		catalogSyncURL: catalogSyncURL,
		idGen:          idGen,
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

	go u.syncToCatalog(v.ShortID)
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

	go u.syncToCatalog(g.ShortID)
	return g, nil
}

func (u *contentUseCase) AddAssets(ctx context.Context, id uuid.UUID, req AddAssetsRequest) error {
	var assets []domain.Asset
	for _, a := range req.Assets {
		assets = append(assets, domain.Asset{
			ID:        uuid.Must(uuid.NewV7()),
			AssetRole: a.Role,
			S3Key:     a.MinIOKey,
		})
	}

	if req.ParentType == "video" {
		if err := u.repo.AddVideoAssets(ctx, id, assets); err != nil {
			return err
		}
	} else {
		if err := u.repo.AddGalleryAssets(ctx, id, assets); err != nil {
			return err
		}
	}

	// We need shortID for sync. Fetch it based on type.
	var shortID string
	if req.ParentType == "video" {
		v, err := u.repo.GetVideoByID(ctx, id)
		if err == nil {
			shortID = v.ShortID
		}
	} else {
		g, err := u.repo.GetGalleryByID(ctx, id)
		if err == nil {
			shortID = g.ShortID
		}
	}

	if shortID != "" {
		go u.syncToCatalog(shortID)
	}

	return nil
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
	go u.syncToCatalog(v.ShortID)
	return nil
}

func (u *contentUseCase) UpdateGallery(ctx context.Context, id uuid.UUID, g *domain.Gallery) error {
	g.ID = id
	if err := u.repo.UpdateGallery(ctx, g); err != nil {
		return err
	}
	go u.syncToCatalog(g.ShortID)
	return nil
}

func (u *contentUseCase) syncToCatalog(shortID string) {
	// Sync is now handled by Benthos polling PostgreSQL view.
	// Manual trigger is disabled to avoid 404/500 issues during transition.
	/*
		if u.catalogSyncURL == "" {
			return
		}
		url := fmt.Sprintf("%s/api/catalog/v1/internal/catalog/sync/%s", u.catalogSyncURL, shortID)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte("{}")))
		if err != nil {
			log.Printf("Failed to trigger catalog sync for %s: %v", shortID, err)
			return
		}
		defer resp.Body.Close()
	*/
}
