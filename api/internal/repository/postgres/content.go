package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/k-wa-wa/pechka/api/internal/domain"
)

type ContentRepository struct {
	pool *pgxpool.Pool
}

func NewContentRepository(pool *pgxpool.Pool) *ContentRepository {
	return &ContentRepository{pool: pool}
}

type CreateContentParams struct {
	ShortID         string
	ContentType     domain.ContentType
	DiscID          *string
	Title           string
	Description     string
	DurationSeconds *int
	Is360           bool
	Tags            []string
	Status          domain.ContentStatus
}

func (r *ContentRepository) Create(ctx context.Context, params CreateContentParams) (*domain.Content, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO contents (short_id, content_type, disc_id, title, description, duration_seconds, is_360, tags, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, short_id, content_type, disc_id, title, description, duration_seconds, is_360, tags, status, published_at, created_at, updated_at
	`, params.ShortID, params.ContentType, params.DiscID, params.Title, params.Description,
		params.DurationSeconds, params.Is360, params.Tags, params.Status)

	return scanContent(row)
}

type UpdateContentParams struct {
	ID          string
	Title       *string
	Description *string
	Tags        []string
	Status      *domain.ContentStatus
}

func (r *ContentRepository) Update(ctx context.Context, params UpdateContentParams) (*domain.Content, error) {
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if params.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *params.Title)
		argIdx++
	}
	if params.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *params.Description)
		argIdx++
	}
	if params.Tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("tags = $%d", argIdx))
		args = append(args, params.Tags)
		argIdx++
	}
	if params.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *params.Status)
		argIdx++
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, params.ID)
	}

	args = append(args, params.ID)
	query := fmt.Sprintf(`
		UPDATE contents SET %s
		WHERE id = $%d
		RETURNING id, short_id, content_type, disc_id, title, description, duration_seconds, is_360, tags, status, published_at, created_at, updated_at
	`, strings.Join(setClauses, ", "), argIdx)

	row := r.pool.QueryRow(ctx, query, args...)
	return scanContent(row)
}

func (r *ContentRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM contents WHERE id = $1", id)
	return err
}

func (r *ContentRepository) GetByID(ctx context.Context, id string) (*domain.Content, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, short_id, content_type, disc_id, title, description, duration_seconds, is_360, tags, status, published_at, created_at, updated_at
		FROM contents WHERE id = $1
	`, id)
	return scanContent(row)
}

type ListContentsParams struct {
	Status *domain.ContentStatus
	Limit  int
	Offset int
}

func (r *ContentRepository) List(ctx context.Context, params ListContentsParams) ([]*domain.Content, error) {
	query := `SELECT id, short_id, content_type, disc_id, title, description, duration_seconds, is_360, tags, status, published_at, created_at, updated_at FROM contents`
	args := []any{}
	argIdx := 1

	if params.Status != nil {
		query += fmt.Sprintf(" WHERE status = $%d", argIdx)
		args = append(args, *params.Status)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY updated_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, params.Limit, params.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contents []*domain.Content
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, err
		}
		contents = append(contents, c)
	}
	return contents, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanContent(row scanner) (*domain.Content, error) {
	var c domain.Content
	err := row.Scan(
		&c.ID, &c.ShortID, &c.ContentType, &c.DiscID, &c.Title, &c.Description,
		&c.DurationSeconds, &c.Is360, &c.Tags, &c.Status, &c.PublishedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
