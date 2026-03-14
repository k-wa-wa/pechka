package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"pechka/streaming-service/api/internal/domain"
)

type DB interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

type contentRepository struct {
	pool *pgxpool.Pool
}

func NewContentRepository(pool *pgxpool.Pool) domain.ContentRepository {
	return &contentRepository{
		pool: pool,
	}
}

// Video operations

func (r *contentRepository) CreateVideo(ctx context.Context, v *domain.Video) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO videos (id, short_id, title, description, rating, is_360, duration_seconds, director, published_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	now := time.Now()
	v.CreatedAt = now
	v.UpdatedAt = now

	_, err = tx.Exec(ctx, query,
		v.ID, v.ShortID, v.Title, v.Description, v.Rating,
		v.Is360, v.DurationSeconds, v.Director,
		v.PublishedAt, v.CreatedAt, v.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert video: %w", err)
	}

	if len(v.Assets) > 0 {
		if err := r.addAssetsTx(ctx, tx, "video_assets", "video_id", v.ID, v.Assets); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *contentRepository) GetVideoByShortID(ctx context.Context, shortID string) (*domain.Video, error) {
	query := `
		SELECT id, short_id, title, description, rating, is_360, duration_seconds, director, published_at, created_at, updated_at
		FROM videos
		WHERE short_id = $1
	`
	var v domain.Video
	err := r.pool.QueryRow(ctx, query, shortID).Scan(
		&v.ID, &v.ShortID, &v.Title, &v.Description, &v.Rating,
		&v.Is360, &v.DurationSeconds, &v.Director,
		&v.PublishedAt, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("video not found: %s", shortID)
		}
		return nil, err
	}

	assets, err := r.getVideoAssets(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	v.Assets = assets
	return &v, nil
}

func (r *contentRepository) GetVideoByID(ctx context.Context, id uuid.UUID) (*domain.Video, error) {
	query := `
		SELECT id, short_id, title, description, rating, is_360, duration_seconds, director, published_at, created_at, updated_at
		FROM videos
		WHERE id = $1
	`
	var v domain.Video
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&v.ID, &v.ShortID, &v.Title, &v.Description, &v.Rating,
		&v.Is360, &v.DurationSeconds, &v.Director,
		&v.PublishedAt, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("video not found: %s", id)
		}
		return nil, err
	}

	assets, err := r.getVideoAssets(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	v.Assets = assets
	return &v, nil
}

func (r *contentRepository) UpdateVideo(ctx context.Context, v *domain.Video) error {
	query := `
		UPDATE videos
		SET title = $1, description = $2, rating = $3, is_360 = $4, duration_seconds = $5, director = $6, published_at = $7, updated_at = $8
		WHERE id = $9
	`
	v.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, query,
		v.Title, v.Description, v.Rating, v.Is360, v.DurationSeconds, v.Director, v.PublishedAt, v.UpdatedAt, v.ID,
	)
	return err
}

func (r *contentRepository) ListVideos(ctx context.Context) ([]*domain.Video, error) {
	query := `SELECT id, short_id, title, description, rating, is_360, duration_seconds, director, published_at, created_at, updated_at FROM videos ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domain.Video
	for rows.Next() {
		var v domain.Video
		if err := rows.Scan(
			&v.ID, &v.ShortID, &v.Title, &v.Description, &v.Rating,
			&v.Is360, &v.DurationSeconds, &v.Director,
			&v.PublishedAt, &v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, err
		}
		
		assets, err := r.getVideoAssets(ctx, v.ID)
		if err != nil {
			return nil, err
		}
		v.Assets = assets
		
		results = append(results, &v)
	}
	return results, nil
}

// Gallery operations

func (r *contentRepository) CreateGallery(ctx context.Context, g *domain.Gallery) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO galleries (id, short_id, title, description, rating, published_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	now := time.Now()
	g.CreatedAt = now
	g.UpdatedAt = now

	_, err = tx.Exec(ctx, query, g.ID, g.ShortID, g.Title, g.Description, g.Rating, g.PublishedAt, g.CreatedAt, g.UpdatedAt)
	if err != nil {
		return err
	}

	if len(g.Assets) > 0 {
		if err := r.addAssetsTx(ctx, tx, "gallery_assets", "gallery_id", g.ID, g.Assets); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *contentRepository) GetGalleryByShortID(ctx context.Context, shortID string) (*domain.Gallery, error) {
	query := `SELECT id, short_id, title, description, rating, published_at, created_at, updated_at FROM galleries WHERE short_id = $1`
	var g domain.Gallery
	err := r.pool.QueryRow(ctx, query, shortID).Scan(&g.ID, &g.ShortID, &g.Title, &g.Description, &g.Rating, &g.PublishedAt, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("gallery not found: %s", shortID)
		}
		return nil, err
	}

	assets, err := r.getGalleryAssets(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	g.Assets = assets
	return &g, nil
}

func (r *contentRepository) GetGalleryByID(ctx context.Context, id uuid.UUID) (*domain.Gallery, error) {
	query := `SELECT id, short_id, title, description, rating, published_at, created_at, updated_at FROM galleries WHERE id = $1`
	var g domain.Gallery
	err := r.pool.QueryRow(ctx, query, id).Scan(&g.ID, &g.ShortID, &g.Title, &g.Description, &g.Rating, &g.PublishedAt, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("gallery not found: %s", id)
		}
		return nil, err
	}

	assets, err := r.getGalleryAssets(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	g.Assets = assets
	return &g, nil
}

func (r *contentRepository) UpdateGallery(ctx context.Context, g *domain.Gallery) error {
	query := `UPDATE galleries SET title = $1, description = $2, rating = $3, published_at = $4, updated_at = $5 WHERE id = $6`
	g.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, query, g.Title, g.Description, g.Rating, g.PublishedAt, g.UpdatedAt, g.ID)
	return err
}

func (r *contentRepository) ListGalleries(ctx context.Context) ([]*domain.Gallery, error) {
	query := `SELECT id, short_id, title, description, rating, published_at, created_at, updated_at FROM galleries ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domain.Gallery
	for rows.Next() {
		var g domain.Gallery
		if err := rows.Scan(&g.ID, &g.ShortID, &g.Title, &g.Description, &g.Rating, &g.PublishedAt, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		
		assets, err := r.getGalleryAssets(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		g.Assets = assets
		
		results = append(results, &g)
	}
	return results, nil
}

// Asset operations

func (r *contentRepository) AddVideoAssets(ctx context.Context, videoID uuid.UUID, assets []domain.Asset) error {
	return r.addAssetsTx(ctx, r.pool, "video_assets", "video_id", videoID, assets)
}

func (r *contentRepository) AddGalleryAssets(ctx context.Context, galleryID uuid.UUID, assets []domain.Asset) error {
	return r.addAssetsTx(ctx, r.pool, "gallery_assets", "gallery_id", galleryID, assets)
}

func (r *contentRepository) addAssetsTx(ctx context.Context, exec DB, tableName, fkName string, parentID uuid.UUID, assets []domain.Asset) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, %s, asset_role, s3_key, public_url) VALUES ($1, $2, $3, $4, $5)`, tableName, fkName)
	for _, a := range assets {
		id := a.ID
		if id == uuid.Nil {
			id = uuid.Must(uuid.NewV7())
		}
		_, err := exec.Exec(ctx, query, id, parentID, a.AssetRole, a.S3Key, a.PublicURL)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *contentRepository) getVideoAssets(ctx context.Context, videoID uuid.UUID) ([]domain.Asset, error) {
	return r.getAssets(ctx, "video_assets", "video_id", videoID)
}

func (r *contentRepository) getGalleryAssets(ctx context.Context, galleryID uuid.UUID) ([]domain.Asset, error) {
	return r.getAssets(ctx, "gallery_assets", "gallery_id", galleryID)
}

func (r *contentRepository) getAssets(ctx context.Context, tableName, fkName string, parentID uuid.UUID) ([]domain.Asset, error) {
	query := fmt.Sprintf(`SELECT id, asset_role, s3_key, public_url FROM %s WHERE %s = $1`, tableName, fkName)
	rows, err := r.pool.Query(ctx, query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []domain.Asset
	for rows.Next() {
		var a domain.Asset
		if err := rows.Scan(&a.ID, &a.AssetRole, &a.S3Key, &a.PublicURL); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, nil
}

func (r *contentRepository) ListAllShortIDs(ctx context.Context) ([]string, error) {
	query := `SELECT short_id FROM videos UNION SELECT short_id FROM galleries`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *contentRepository) CheckDuplicateByS3Key(ctx context.Context, s3Key string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM (
				SELECT s3_key FROM video_assets WHERE s3_key = $1
				UNION ALL
				SELECT s3_key FROM gallery_assets WHERE s3_key = $1
			) t
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, s3Key).Scan(&exists)
	return exists, err
}
