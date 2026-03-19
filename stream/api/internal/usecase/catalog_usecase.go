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
	Search(ctx context.Context, query string, tags []string) ([]*domain.CatalogContent, error)
	SyncContent(ctx context.Context, shortID string, metaRepo domain.ContentRepository) error
}

type catalogUseCase struct {
	repo       domain.CatalogRepository
	searchRepo domain.SearchRepository
}

func NewCatalogUseCase(repo domain.CatalogRepository, searchRepo domain.SearchRepository) CatalogUseCase {
	return &catalogUseCase{
		repo:       repo,
		searchRepo: searchRepo,
	}
}

func (u *catalogUseCase) GetHome(ctx context.Context) (interface{}, error) {
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

	return res, nil
}

func (u *catalogUseCase) GetContentDetails(ctx context.Context, shortID string) (*domain.CatalogContent, error) {
	res, err := u.repo.GetByShortID(ctx, shortID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (u *catalogUseCase) Search(ctx context.Context, query string, tags []string) ([]*domain.CatalogContent, error) {
	ids, err := u.searchRepo.SearchIDs(ctx, query, tags)
	if err != nil {
		// Log error and fallback to Mongo search if needed, but for now just return error
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(ids) == 0 {
		return []*domain.CatalogContent{}, nil
	}

	contents, err := u.repo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contents from storage: %w", err)
	}

	// Reorder results to match Elasticsearch ranking
	contentMap := make(map[string]*domain.CatalogContent)
	for _, c := range contents {
		contentMap[c.ID] = c
	}

	ordered := make([]*domain.CatalogContent, 0, len(ids))
	for _, id := range ids {
		if c, ok := contentMap[id]; ok {
			ordered = append(ordered, c)
		}
	}

	return ordered, nil
}

func (u *catalogUseCase) SyncContent(ctx context.Context, shortID string, metaRepo domain.ContentRepository) error {
	// 1. Try to fetch as Video
	video, videoErr := metaRepo.GetVideoByShortID(ctx, shortID)
	
	var catalog *domain.CatalogContent
	if videoErr == nil {
		catalog = &domain.CatalogContent{
			ID:          video.ID.String(),
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
			Tags:      video.Tags,
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
			ID:          gallery.ID.String(),
			ShortID:     gallery.ShortID,
			Type:        domain.ContentTypeImageGallery,
			Title:       gallery.Title,
			Description: gallery.Description,
			Rating:      func() float64 { if gallery.Rating != nil { return *gallery.Rating }; return 0 }(),
			Metadata:    make(map[string]interface{}),
			Assets:      make(map[string]string),
			Tags:        gallery.Tags,
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

	return nil
}
