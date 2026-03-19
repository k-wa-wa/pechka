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

// SyncContent fetches the latest data from PostgreSQL (via metaRepo) and upserts it to MongoDB.
func (u *catalogUseCase) SyncContent(ctx context.Context, shortID string, metaRepo domain.ContentRepository) error {
	c, err := metaRepo.GetContentByShortID(ctx, shortID)
	if err != nil {
		return fmt.Errorf("content not found in metadata for shortID %s: %w", shortID, err)
	}

	catalog := contentToCatalog(c)

	if err := u.repo.Upsert(ctx, catalog); err != nil {
		return err
	}
	return nil
}

// contentToCatalog converts a PostgreSQL Content to a denormalized CatalogContent for MongoDB.
func contentToCatalog(c *domain.Content) *domain.CatalogContent {
	rating := 0.0
	if c.Rating != nil {
		rating = *c.Rating
	}

	metadata := map[string]interface{}{}
	if c.VideoDetails != nil {
		metadata["director"] = c.VideoDetails.Director
		metadata["is_360"] = c.VideoDetails.Is360
		metadata["duration_seconds"] = c.VideoDetails.DurationSeconds
	}

	assets := make(map[string]string)
	for _, a := range c.Assets {
		val := a.S3Key
		if a.PublicURL != "" {
			val = a.PublicURL
		}
		assets[string(a.AssetRole)] = val
	}

	tags := c.Tags
	if tags == nil {
		tags = []string{}
	}

	return &domain.CatalogContent{
		ID:          c.ID.String(),
		ShortID:     c.ShortID,
		Type:        c.ContentType,
		Title:       c.Title,
		Description: c.Description,
		Rating:      rating,
		Metadata:    metadata,
		Assets:      assets,
		Tags:        tags,
		UpdatedAt:   time.Now(),
	}
}
