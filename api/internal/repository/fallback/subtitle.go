package fallback

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	pgRepo "github.com/k-wa-wa/pechka/api/internal/repository/postgres"
)

// SubtitlePrimaryRepository represents the primary (MongoDB) interface for subtitles.
type SubtitlePrimaryRepository interface {
	GetByShortIDAndLanguage(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error)
}

// SubtitleFallbackStore represents the fallback (PostgreSQL) query interface for subtitles.
type SubtitleFallbackStore interface {
	GetByShortIDAndLanguage(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error)
}

// FallbackSubtitleRepository wraps primary subtitle repository with PostgreSQL fallback and circuit breaking.
type FallbackSubtitleRepository struct {
	primary  SubtitlePrimaryRepository
	fallback SubtitleFallbackStore
	breaker  *CircuitBreaker
}

// NewSubtitleRepository creates a new FallbackSubtitleRepository with default PostgresSubtitleStore.
func NewSubtitleRepository(primary SubtitlePrimaryRepository, pool *pgxpool.Pool, breaker *CircuitBreaker) *FallbackSubtitleRepository {
	return NewSubtitleRepositoryWithStore(primary, NewPostgresSubtitleStore(pool), breaker)
}

// NewSubtitleRepositoryWithStore creates a new FallbackSubtitleRepository with a custom fallback store.
func NewSubtitleRepositoryWithStore(primary SubtitlePrimaryRepository, fallbackStore SubtitleFallbackStore, breaker *CircuitBreaker) *FallbackSubtitleRepository {
	return &FallbackSubtitleRepository{
		primary:  primary,
		fallback: fallbackStore,
		breaker:  breaker,
	}
}

// GetByShortIDAndLanguage retrieves subtitles for the specified shortID and language, falling back to PostgreSQL if MongoDB fails.
func (r *FallbackSubtitleRepository) GetByShortIDAndLanguage(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
	return Execute(
		ctx,
		r.breaker,
		"get_subtitle",
		"mongo",
		func(c context.Context) (*domain.MongoSubtitle, error) {
			return r.primary.GetByShortIDAndLanguage(c, shortID, language)
		},
		func(c context.Context) (*domain.MongoSubtitle, error) {
			return r.fallback.GetByShortIDAndLanguage(c, shortID, language)
		},
		isMongoFailure,
	)
}

// PostgresSubtitleStore executes direct PostgreSQL fallback queries for subtitles.
type PostgresSubtitleStore struct {
	pool *pgxpool.Pool
}

// NewPostgresSubtitleStore creates a new PostgresSubtitleStore.
func NewPostgresSubtitleStore(pool *pgxpool.Pool) *PostgresSubtitleStore {
	return &PostgresSubtitleStore{pool: pool}
}

// BuildSubtitleQuery returns the query for fetching published subtitles with cues.
func BuildSubtitleQuery() string {
	return `
SELECT
    c.short_id,
    t.language,
    t.status,
    COALESCE(
        (SELECT json_agg(json_build_object(
            'start_ms', cu.start_ms,
            'end_ms', cu.end_ms,
            'text', cu.text
         ) ORDER BY cu.seq, cu.start_ms, cu.id)
         FROM subtitle_cues cu
         WHERE cu.track_id = t.id
        ), '[]'::json
    ) AS cues_json
FROM subtitle_tracks t
JOIN contents c ON c.id = t.content_id
WHERE c.short_id = $1 AND t.language = $2 AND t.status = 'published' AND c.archived_at IS NULL AND c.deleted_at IS NULL
`
}

// GetByShortIDAndLanguage executes the fallback query for subtitles against PostgreSQL.
func (s *PostgresSubtitleStore) GetByShortIDAndLanguage(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
	if s.pool == nil {
		return nil, errors.New("postgres connection pool is nil")
	}

	query := BuildSubtitleQuery()

	return pgRepo.WithRetry(ctx, s.pool, func() (*domain.MongoSubtitle, error) {
		var (
			subShortID  string
			subLanguage string
			subStatus   string
			rawCues     []byte
		)

		err := s.pool.QueryRow(ctx, query, shortID, language).Scan(
			&subShortID,
			&subLanguage,
			&subStatus,
			&rawCues,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, mongo.ErrNoDocuments
			}
			return nil, err
		}

		var cues []domain.MongoCue
		if len(rawCues) > 0 {
			if err := json.Unmarshal(rawCues, &cues); err != nil {
				return nil, fmt.Errorf("failed to unmarshal cues json: %w", err)
			}
		}
		if cues == nil {
			cues = []domain.MongoCue{}
		}

		return &domain.MongoSubtitle{
			ID:       subShortID + ":" + subLanguage,
			ShortID:  subShortID,
			Language: subLanguage,
			Status:   subStatus,
			Cues:     cues,
		}, nil
	})
}
