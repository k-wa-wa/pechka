package fallback

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	elasticRepo "github.com/k-wa-wa/pechka/api/internal/repository/elastic"
	pgRepo "github.com/k-wa-wa/pechka/api/internal/repository/postgres"
)

// SearchPrimaryRepository represents the primary (Elasticsearch) search interface.
type SearchPrimaryRepository interface {
	Search(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error)
}

// SearchFallbackStore represents the fallback (PostgreSQL) search query interface.
type SearchFallbackStore interface {
	Search(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error)
}

// FallbackSearchRepository wraps Elasticsearch search repository with PostgreSQL fallback and circuit breaking.
type FallbackSearchRepository struct {
	primary  SearchPrimaryRepository
	fallback SearchFallbackStore
	breaker  *CircuitBreaker
}

// NewSearchRepository creates a new FallbackSearchRepository with default PostgresSearchStore.
func NewSearchRepository(primary SearchPrimaryRepository, pool *pgxpool.Pool, breaker *CircuitBreaker) *FallbackSearchRepository {
	return NewSearchRepositoryWithStore(primary, NewPostgresSearchStore(pool), breaker)
}

// NewSearchRepositoryWithStore creates a new FallbackSearchRepository with a custom fallback store.
func NewSearchRepositoryWithStore(primary SearchPrimaryRepository, fallbackStore SearchFallbackStore, breaker *CircuitBreaker) *FallbackSearchRepository {
	return &FallbackSearchRepository{
		primary:  primary,
		fallback: fallbackStore,
		breaker:  breaker,
	}
}

func isElasticFailure(err error) bool {
	if err == nil {
		return false
	}
	// 400 Bad Request 等のクライアント入力起因エラーはインフラ障害とみなさない
	msg := err.Error()
	if strings.Contains(msg, "[400") || strings.Contains(msg, "400 Bad Request") || strings.Contains(msg, "status: 400") {
		return false
	}
	return true
}

// Search searches contents, falling back to PostgreSQL ILIKE query if Elasticsearch fails or breaker is open.
func (r *FallbackSearchRepository) Search(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error) {
	// Sanitize client-supplied parameters
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset > 10000 {
		offset = 10000
	}

	return Execute(
		ctx,
		r.breaker,
		"search",
		"elastic",
		func(c context.Context) ([]*elasticRepo.SearchResult, error) {
			return r.primary.Search(c, query, limit, offset)
		},
		func(c context.Context) ([]*elasticRepo.SearchResult, error) {
			return r.fallback.Search(c, query, limit, offset)
		},
		isElasticFailure,
	)
}

// PostgresSearchStore executes fallback search queries against PostgreSQL.
type PostgresSearchStore struct {
	pool *pgxpool.Pool
}

// NewPostgresSearchStore creates a new PostgresSearchStore.
func NewPostgresSearchStore(pool *pgxpool.Pool) *PostgresSearchStore {
	return &PostgresSearchStore{pool: pool}
}

// BuildSearchQuery constructs the SQL query and parameters for fallback search.
func BuildSearchQuery(query string, limit, offset int) (string, []any) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	sql := `
SELECT
    c.short_id,
    COALESCE(c.title, '') AS title,
    COALESCE(c.description, '') AS description,
    c.content_type,
    c.channel_id,
    COALESCE(ch.name, c.channel_id) AS channel_name,
    COALESCE(c.tags, '{}') AS tags,
    c.status
FROM contents c
LEFT JOIN channels ch ON c.channel_id = ch.id
WHERE c.archived_at IS NULL
  AND (
      c.title ILIKE $1
      OR c.description ILIKE $1
      OR EXISTS (SELECT 1 FROM unnest(c.tags) t WHERE t ILIKE $1)
  )
ORDER BY c.updated_at DESC
LIMIT $2 OFFSET $3
`
	pattern := "%" + query + "%"
	args := []any{pattern, limit, offset}
	return sql, args
}

// Search executes ILIKE search against PostgreSQL.
func (s *PostgresSearchStore) Search(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error) {
	if s.pool == nil {
		return nil, errors.New("postgres connection pool is nil")
	}

	sql, args := BuildSearchQuery(query, limit, offset)

	return pgRepo.WithRetry(ctx, s.pool, func() ([]*elasticRepo.SearchResult, error) {
		rows, err := s.pool.Query(ctx, sql, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var results []*elasticRepo.SearchResult
		for rows.Next() {
			var (
				item        elasticRepo.SearchResult
				contentType string
				status      string
				tags        []string
			)

			err := rows.Scan(
				&item.ShortID,
				&item.Title,
				&item.Description,
				&contentType,
				&item.ChannelID,
				&item.ChannelName,
				&tags,
				&status,
			)
			if err != nil {
				return nil, err
			}

			item.ContentType = domain.ContentType(contentType)
			item.Status = domain.ContentStatus(status)
			if tags == nil {
				tags = []string{}
			}
			item.Tags = tags

			results = append(results, &item)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if results == nil {
			results = []*elasticRepo.SearchResult{}
		}
		return results, nil
	})
}
