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
	return &contentRepository{pool: pool}
}

// ─── Content CRUD ─────────────────────────────────────────────────────────────

func (r *contentRepository) CreateContent(ctx context.Context, c *domain.Content) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	c.CreatedAt = now

	_, err = tx.Exec(ctx, `
		INSERT INTO contents (id, short_id, content_type, title, description, rating, tags, visibility, allowed_groups, published_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, c.ID, c.ShortID, c.ContentType, c.Title, c.Description, c.Rating, c.Tags, c.Visibility, c.AllowedGroups,
		c.PublishedAt, c.CreatedAt, c.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert content: %w", err)
	}

	// video / vr360 のみ content_videos に書き込む
	if c.VideoDetails != nil {
		_, err = tx.Exec(ctx, `
			INSERT INTO content_videos (content_id, is_360, duration_seconds, director)
			VALUES ($1, $2, $3, $4)
		`, c.ID, c.VideoDetails.Is360, c.VideoDetails.DurationSeconds, c.VideoDetails.Director)
		if err != nil {
			return fmt.Errorf("failed to insert content_videos: %w", err)
		}
	}

	if len(c.Assets) > 0 {
		if err := r.addAssetsTx(ctx, tx, c.ID, c.Assets); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *contentRepository) GetContentByShortID(ctx context.Context, shortID string) (*domain.Content, error) {
	return r.getContent(ctx, "short_id", shortID)
}

func (r *contentRepository) GetContentByID(ctx context.Context, id uuid.UUID) (*domain.Content, error) {
	return r.getContent(ctx, "id", id)
}

// getContent fetches a Content by a column name and value, joining content_videos.
func (r *contentRepository) getContent(ctx context.Context, col string, val interface{}) (*domain.Content, error) {
	query := fmt.Sprintf(`
		SELECT
			c.id, c.short_id, c.content_type, c.title, c.description, c.rating, c.tags,
			c.visibility, c.allowed_groups,
			c.published_at, c.created_at, c.updated_at,
			cv.is_360, cv.duration_seconds, cv.director
		FROM contents c
		LEFT JOIN content_videos cv ON cv.content_id = c.id
		WHERE c.%s = $1
	`, col)

	row := r.pool.QueryRow(ctx, query, val)
	c, err := scanContent(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("content not found: %v", val)
		}
		return nil, err
	}

	assets, err := r.getAssets(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	c.Assets = assets
	return c, nil
}

func (r *contentRepository) UpdateContent(ctx context.Context, c *domain.Content) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE contents
		SET title = $1, description = $2, rating = $3, tags = $4, visibility = $5, allowed_groups = $6, published_at = $7
		WHERE id = $8
	`, c.Title, c.Description, c.Rating, c.Tags, c.Visibility, c.AllowedGroups, c.PublishedAt, c.ID)
	if err != nil {
		return fmt.Errorf("failed to update content: %w", err)
	}

	if c.VideoDetails != nil {
		// UPSERT: insert or update content_videos
		_, err = tx.Exec(ctx, `
			INSERT INTO content_videos (content_id, is_360, duration_seconds, director)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (content_id) DO UPDATE
			SET is_360 = EXCLUDED.is_360,
			    duration_seconds = EXCLUDED.duration_seconds,
			    director = EXCLUDED.director
		`, c.ID, c.VideoDetails.Is360, c.VideoDetails.DurationSeconds, c.VideoDetails.Director)
		if err != nil {
			return fmt.Errorf("failed to upsert content_videos: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *contentRepository) ListContents(ctx context.Context) ([]*domain.Content, error) {
	query := `
		SELECT
			c.id, c.short_id, c.content_type, c.title, c.description, c.rating, c.tags,
			c.visibility, c.allowed_groups,
			c.published_at, c.created_at, c.updated_at,
			cv.is_360, cv.duration_seconds, cv.director
		FROM contents c
		LEFT JOIN content_videos cv ON cv.content_id = c.id
		ORDER BY c.created_at DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domain.Content
	for rows.Next() {
		c, err := scanContentRow(rows)
		if err != nil {
			return nil, err
		}
		assets, err := r.getAssets(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		c.Assets = assets
		results = append(results, c)
	}
	return results, nil
}

// ─── Asset Operations ─────────────────────────────────────────────────────────

func (r *contentRepository) AddAssets(ctx context.Context, contentID uuid.UUID, assets []domain.Asset) error {
	return r.addAssetsTx(ctx, r.pool, contentID, assets)
}

func (r *contentRepository) addAssetsTx(ctx context.Context, exec DB, contentID uuid.UUID, assets []domain.Asset) error {
	for _, a := range assets {
		id := a.ID
		if id == uuid.Nil {
			id = uuid.Must(uuid.NewV7())
		}
		_, err := exec.Exec(ctx,
			`INSERT INTO assets (id, content_id, asset_role, s3_key, public_url) VALUES ($1, $2, $3, $4, $5)`,
			id, contentID, a.AssetRole, a.S3Key, a.PublicURL,
		)
		if err != nil {
			return fmt.Errorf("failed to insert asset: %w", err)
		}
	}
	return nil
}

func (r *contentRepository) getAssets(ctx context.Context, contentID uuid.UUID) ([]domain.Asset, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, asset_role, s3_key, public_url FROM assets WHERE content_id = $1`, contentID)
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

// ─── Utility ─────────────────────────────────────────────────────────────────

func (r *contentRepository) CheckDuplicateByS3Key(ctx context.Context, s3Key string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM assets WHERE s3_key = $1)`, s3Key,
	).Scan(&exists)
	return exists, err
}

func (r *contentRepository) ListAllShortIDs(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT short_id FROM contents`)
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

// ─── Scan helpers ─────────────────────────────────────────────────────────────

// scanContent scans a single row returned by QueryRow (contents LEFT JOIN content_videos).
func scanContent(row pgx.Row) (*domain.Content, error) {
	var c domain.Content
	var is360 *bool
	var durationSec *int
	var director *string

	err := row.Scan(
		&c.ID, &c.ShortID, &c.ContentType, &c.Title, &c.Description, &c.Rating, &c.Tags,
		&c.Visibility, &c.AllowedGroups,
		&c.PublishedAt, &c.CreatedAt, &c.UpdatedAt,
		&is360, &durationSec, &director,
	)
	if err != nil {
		return nil, err
	}

	if is360 != nil || durationSec != nil || director != nil {
		c.VideoDetails = &domain.VideoDetails{
			Is360:           boolVal(is360),
			DurationSeconds: intVal(durationSec),
			Director:        strVal(director),
		}
	}
	return &c, nil
}

// scanContentRow scans a row from Rows (used in ListContents).
func scanContentRow(rows pgx.Rows) (*domain.Content, error) {
	var c domain.Content
	var is360 *bool
	var durationSec *int
	var director *string

	err := rows.Scan(
		&c.ID, &c.ShortID, &c.ContentType, &c.Title, &c.Description, &c.Rating, &c.Tags,
		&c.Visibility, &c.AllowedGroups,
		&c.PublishedAt, &c.CreatedAt, &c.UpdatedAt,
		&is360, &durationSec, &director,
	)
	if err != nil {
		return nil, err
	}

	if is360 != nil || durationSec != nil || director != nil {
		c.VideoDetails = &domain.VideoDetails{
			Is360:           boolVal(is360),
			DurationSeconds: intVal(durationSec),
			Director:        strVal(director),
		}
	}
	return &c, nil
}

func boolVal(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

func intVal(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func strVal(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
