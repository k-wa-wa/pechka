package fallback

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	mongoRepo "github.com/k-wa-wa/pechka/api/internal/repository/mongo"
	pgRepo "github.com/k-wa-wa/pechka/api/internal/repository/postgres"
)

// ContentPrimaryRepository represents the primary (MongoDB) interface.
type ContentPrimaryRepository interface {
	List(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error)
	GetByShortID(ctx context.Context, shortID string) (*domain.MongoContent, error)
	GetVariantsByShortID(ctx context.Context, shortID string) ([]domain.MongoVariant, error)
}

// ContentFallbackStore represents the fallback (PostgreSQL) query interface.
type ContentFallbackStore interface {
	List(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error)
	GetByShortID(ctx context.Context, shortID string) (*domain.MongoContent, error)
	GetVariantsByShortID(ctx context.Context, shortID string) ([]domain.MongoVariant, error)
}

// FallbackContentRepository wraps primary MongoDB repository with PostgreSQL fallback and circuit breaking.
type FallbackContentRepository struct {
	primary  ContentPrimaryRepository
	fallback ContentFallbackStore
	breaker  *CircuitBreaker
}

// NewContentRepository creates a new FallbackContentRepository with default PostgresContentStore.
func NewContentRepository(primary ContentPrimaryRepository, pool *pgxpool.Pool, breaker *CircuitBreaker) *FallbackContentRepository {
	return NewContentRepositoryWithStore(primary, NewPostgresContentStore(pool), breaker)
}

// NewContentRepositoryWithStore creates a new FallbackContentRepository with a custom fallback store.
func NewContentRepositoryWithStore(primary ContentPrimaryRepository, fallbackStore ContentFallbackStore, breaker *CircuitBreaker) *FallbackContentRepository {
	return &FallbackContentRepository{
		primary:  primary,
		fallback: fallbackStore,
		breaker:  breaker,
	}
}

func isMongoFailure(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false
	}
	return true
}

// List retrieves a list of contents, falling back to PostgreSQL if MongoDB fails or breaker is open.
func (r *FallbackContentRepository) List(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error) {
	// Sanitize client-supplied offset to prevent invalid negative skip errors from crashing the breaker.
	if params.Offset < 0 {
		params.Offset = 0
	}
	if params.Limit <= 0 || params.Limit > 100 {
		params.Limit = 20
	}

	return Execute(
		ctx,
		r.breaker,
		"list",
		"mongo",
		func(c context.Context) ([]*domain.MongoContent, error) {
			return r.primary.List(c, params)
		},
		func(c context.Context) ([]*domain.MongoContent, error) {
			return r.fallback.List(c, params)
		},
		isMongoFailure,
	)
}

// GetByShortID retrieves content by shortID, falling back to PostgreSQL if MongoDB fails.
func (r *FallbackContentRepository) GetByShortID(ctx context.Context, shortID string) (*domain.MongoContent, error) {
	return Execute(
		ctx,
		r.breaker,
		"get",
		"mongo",
		func(c context.Context) (*domain.MongoContent, error) {
			return r.primary.GetByShortID(c, shortID)
		},
		func(c context.Context) (*domain.MongoContent, error) {
			return r.fallback.GetByShortID(c, shortID)
		},
		isMongoFailure,
	)
}

// GetVariantsByShortID retrieves variants by shortID, falling back to PostgreSQL if MongoDB fails.
func (r *FallbackContentRepository) GetVariantsByShortID(ctx context.Context, shortID string) ([]domain.MongoVariant, error) {
	return Execute(
		ctx,
		r.breaker,
		"get_variants",
		"mongo",
		func(c context.Context) ([]domain.MongoVariant, error) {
			return r.primary.GetVariantsByShortID(c, shortID)
		},
		func(c context.Context) ([]domain.MongoVariant, error) {
			return r.fallback.GetVariantsByShortID(c, shortID)
		},
		isMongoFailure,
	)
}

// PostgresContentStore executes direct PostgreSQL fallback queries.
type PostgresContentStore struct {
	pool *pgxpool.Pool
}

// NewPostgresContentStore creates a new PostgresContentStore.
func NewPostgresContentStore(pool *pgxpool.Pool) *PostgresContentStore {
	return &PostgresContentStore{pool: pool}
}

const baseContentSelect = `
SELECT
    c.short_id,
    c.content_type,
    c.channel_id,
    COALESCE(ch.name, c.channel_id) AS channel_name,
    COALESCE(c.title, '') AS title,
    COALESCE(c.description, '') AS description,
    c.duration_seconds,
    COALESCE(c.is_360, false) AS is_360,
    COALESCE(c.tags, '{}') AS tags,
    c.status,
    d.label AS disc_label,
    COALESCE(
        (SELECT json_agg(json_build_object(
            'variant_type', v.variant_type,
            'hls_key', v.hls_key,
            'bandwidth', v.bandwidth,
            'resolution', v.resolution,
            'codecs', v.codecs
         ))
         FROM video_variants v
         WHERE v.content_id = c.id
        ), '[]'::json
    ) AS variants_json,
    c.thumbnail_key,
    c.document_key,
    c.document_mime_type,
    c.published_at,
    c.updated_at,
    EXISTS(
        SELECT 1 FROM subtitle_tracks st
        WHERE st.content_id = c.id AND st.status = 'published'
    ) AS has_subtitles
FROM contents c
LEFT JOIN channels ch ON c.channel_id = ch.id
LEFT JOIN discs d ON c.disc_id = d.id
`

// BuildContentListQuery constructs SQL query and arguments for ListParams.
func BuildContentListQuery(params mongoRepo.ListParams) (string, []any) {
	conditions := []string{"c.archived_at IS NULL"}
	args := []any{}
	argIdx := 1

	if params.ContentType != nil {
		conditions = append(conditions, fmt.Sprintf("c.content_type = $%d", argIdx))
		args = append(args, *params.ContentType)
		argIdx++
	}
	if params.ChannelID != nil {
		conditions = append(conditions, fmt.Sprintf("c.channel_id = $%d", argIdx))
		args = append(args, *params.ChannelID)
		argIdx++
	}
	if params.Status != nil {
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", argIdx))
		args = append(args, *params.Status)
		argIdx++
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	query := baseContentSelect +
		" WHERE " + strings.Join(conditions, " AND ") +
		fmt.Sprintf(" ORDER BY c.updated_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	return query, args
}

// List executes fallback query for contents list against PostgreSQL.
func (s *PostgresContentStore) List(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error) {
	if s.pool == nil {
		return nil, errors.New("postgres connection pool is nil")
	}

	query, args := BuildContentListQuery(params)

	return pgRepo.WithRetry(ctx, s.pool, func() ([]*domain.MongoContent, error) {
		rows, err := s.pool.Query(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var results []*domain.MongoContent
		for rows.Next() {
			c, err := scanMongoContent(rows)
			if err != nil {
				return nil, err
			}
			results = append(results, c)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if results == nil {
			results = []*domain.MongoContent{}
		}
		return results, nil
	})
}

// GetByShortID executes fallback query for single content against PostgreSQL.
func (s *PostgresContentStore) GetByShortID(ctx context.Context, shortID string) (*domain.MongoContent, error) {
	if s.pool == nil {
		return nil, errors.New("postgres connection pool is nil")
	}

	query := baseContentSelect + " WHERE c.short_id = $1 AND c.archived_at IS NULL"

	return pgRepo.WithRetry(ctx, s.pool, func() (*domain.MongoContent, error) {
		row := s.pool.QueryRow(ctx, query, shortID)
		c, err := scanMongoContent(row)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, mongo.ErrNoDocuments
			}
			return nil, err
		}
		return c, nil
	})
}

// GetVariantsByShortID executes fallback query for content variants against PostgreSQL.
func (s *PostgresContentStore) GetVariantsByShortID(ctx context.Context, shortID string) ([]domain.MongoVariant, error) {
	if s.pool == nil {
		return nil, errors.New("postgres connection pool is nil")
	}

	query := `
SELECT
    COALESCE(
        (SELECT json_agg(json_build_object(
            'variant_type', v.variant_type,
            'hls_key', v.hls_key,
            'bandwidth', v.bandwidth,
            'resolution', v.resolution,
            'codecs', v.codecs
         ))
         FROM video_variants v
         WHERE v.content_id = c.id
        ), '[]'::json
    ) AS variants_json
FROM contents c
WHERE c.short_id = $1 AND c.archived_at IS NULL
`

	return pgRepo.WithRetry(ctx, s.pool, func() ([]domain.MongoVariant, error) {
		var rawVariants []byte
		err := s.pool.QueryRow(ctx, query, shortID).Scan(&rawVariants)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, mongo.ErrNoDocuments
			}
			return nil, err
		}

		var variants []domain.MongoVariant
		if len(rawVariants) > 0 {
			if err := json.Unmarshal(rawVariants, &variants); err != nil {
				return nil, fmt.Errorf("failed to unmarshal variants json: %w", err)
			}
		}
		if variants == nil {
			variants = []domain.MongoVariant{}
		}
		return variants, nil
	})
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMongoContent(row rowScanner) (*domain.MongoContent, error) {
	var c domain.MongoContent
	var rawVariants []byte

	err := row.Scan(
		&c.ID,
		&c.ContentType,
		&c.ChannelID,
		&c.ChannelName,
		&c.Title,
		&c.Description,
		&c.DurationSeconds,
		&c.Is360,
		&c.Tags,
		&c.Status,
		&c.DiscLabel,
		&rawVariants,
		&c.ThumbnailKey,
		&c.DocumentKey,
		&c.DocumentMimeType,
		&c.PublishedAt,
		&c.UpdatedAt,
		&c.HasSubtitles,
	)
	if err != nil {
		return nil, err
	}

	if c.Tags == nil {
		c.Tags = []string{}
	}

	if len(rawVariants) > 0 {
		var variants []domain.MongoVariant
		if err := json.Unmarshal(rawVariants, &variants); err != nil {
			return nil, fmt.Errorf("failed to unmarshal variants json: %w", err)
		}
		c.Variants = variants
	}
	if c.Variants == nil {
		c.Variants = []domain.MongoVariant{}
	}

	return &c, nil
}
