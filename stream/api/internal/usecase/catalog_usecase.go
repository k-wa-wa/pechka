package usecase

import (
	"context"
	"fmt"
	"time"

	"pechka/streaming-service/api/internal/domain"
)

type CatalogUseCase interface {
	GetHome(ctx context.Context) (interface{}, error)
	GetContentDetails(ctx context.Context, shortID string) (*domain.CatalogContent, error)
	SyncContent(ctx context.Context, shortID string, metaRepo domain.ContentRepository) error
}

type catalogUseCase struct {
	repo  domain.CatalogRepository
	cache domain.CacheRepository
}

func NewCatalogUseCase(repo domain.CatalogRepository, cache domain.CacheRepository) CatalogUseCase {
	return &catalogUseCase{
		repo:  repo,
		cache: cache,
	}
}

func (u *catalogUseCase) GetHome(ctx context.Context) (interface{}, error) {
	cacheKey := "catalog:home"
	var homeData interface{}
	
	found, err := u.cache.Get(ctx, cacheKey, &homeData)
	if err == nil && found {
		return homeData, nil
	}

	contents, err := u.repo.Search(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contents for home: %w", err)
	}

	res := map[string]interface{}{
		"banners": contents,
		"sections": []interface{}{
			map[string]interface{}{
				"title": "Explore",
				"items": contents,
			},
		},
	}

	_ = u.cache.Set(ctx, cacheKey, res, 1*time.Minute)
	return res, nil
}

func (u *catalogUseCase) GetContentDetails(ctx context.Context, shortID string) (*domain.CatalogContent, error) {
	cacheKey := fmt.Sprintf("content:%s", shortID)
	var content domain.CatalogContent
	
	found, err := u.cache.Get(ctx, cacheKey, &content)
	if err == nil && found {
		return &content, nil
	}

	res, err := u.repo.GetByShortID(ctx, shortID)
	if err != nil {
		return nil, err
	}

	_ = u.cache.Set(ctx, cacheKey, res, 10*time.Minute)
	return res, nil
}

func (u *catalogUseCase) SyncContent(ctx context.Context, shortID string, metaRepo domain.ContentRepository) error {
	// 1. Try to fetch as Video
	video, videoErr := metaRepo.GetVideoByShortID(ctx, shortID)
	
	var catalog *domain.CatalogContent
	if videoErr == nil {
		catalog = &domain.CatalogContent{
			ID:          video.ID,
			ShortID:     video.ShortID,
			Type:        domain.ContentTypeVideo,
			Title:       video.Title,
			Description: video.Description,
			Rating:      func() float64 { if video.Rating != nil { return *video.Rating }; return 0 }(),
			Metadata: map[string]interface{}{
				"director":         video.Director,
				"is_360":           video.Is360,
				"duration_seconds": video.DurationSeconds,
			},
			Assets:    make(map[string]string),
			UpdatedAt: time.Now(),
		}
		for _, asset := range video.Assets {
			val := asset.S3Key
			if asset.PublicURL != "" {
				val = asset.PublicURL
			}
			catalog.Assets[string(asset.AssetRole)] = val
		}
	} else {
		// 2. Try to fetch as Gallery
		gallery, galleryErr := metaRepo.GetGalleryByShortID(ctx, shortID)
		if galleryErr != nil {
			return fmt.Errorf("content not found in metadata for shortID %s: video_err=%v, gallery_err=%v", shortID, videoErr, galleryErr)
		}
		catalog = &domain.CatalogContent{
			ID:          gallery.ID,
			ShortID:     gallery.ShortID,
			Type:        domain.ContentTypeImageGallery,
			Title:       gallery.Title,
			Description: gallery.Description,
			Rating:      func() float64 { if gallery.Rating != nil { return *gallery.Rating }; return 0 }(),
			Metadata:    make(map[string]interface{}),
			Assets:      make(map[string]string),
			UpdatedAt:   time.Now(),
		}
		for _, asset := range gallery.Assets {
			val := asset.S3Key
			if asset.PublicURL != "" {
				val = asset.PublicURL
			}
			catalog.Assets[string(asset.AssetRole)] = val
		}
	}

	// 3. Update Mongo
	if err := u.repo.Upsert(ctx, catalog); err != nil {
		return err
	}

	// 4. Invalidate Cache
	_ = u.cache.Delete(ctx, fmt.Sprintf("content:%s", shortID))
	_ = u.cache.Delete(ctx, "catalog:home")

	return nil
}
