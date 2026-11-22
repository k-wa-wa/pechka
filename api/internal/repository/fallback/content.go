package fallback

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

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
	ListAdmin(ctx context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error)
}

// ContentFallbackStore represents the fallback (PostgreSQL) query interface.
type ContentFallbackStore interface {
	List(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error)
	GetByShortID(ctx context.Context, shortID string) (*domain.MongoContent, error)
	GetVariantsByShortID(ctx context.Context, shortID string) ([]domain.MongoVariant, error)
	ListAdmin(ctx context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error)
}

// FallbackContentRepository wraps primary MongoDB repository with PostgreSQL fallback and circuit breaking.
type FallbackContentRepository struct {
	primary      ContentPrimaryRepository
	fallback     ContentFallbackStore
	breaker      *CircuitBreaker
	adminBreaker *CircuitBreaker
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

// WithAdminBreaker sets a dedicated CircuitBreaker for ListAdmin so admin queries cannot trip the public breaker.
func (r *FallbackContentRepository) WithAdminBreaker(breaker *CircuitBreaker) *FallbackContentRepository {
	r.adminBreaker = breaker
	return r
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

// ListAdmin retrieves admin contents from MongoDB, falling back to PostgreSQL if MongoDB fails or read model is unmigrated.
func (r *FallbackContentRepository) ListAdmin(ctx context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
	if params.Limit <= 0 || params.Limit > 100 {
		params.Limit = 20
	}
	if params.Sort == "" {
		params.Sort = domain.ContentSortUpdatedDesc
	}

	breaker := r.adminBreaker
	if breaker == nil {
		breaker = r.breaker
	}

	return Execute(
		ctx,
		breaker,
		"list_admin",
		"mongo",
		func(c context.Context) (*domain.AdminContentPage, error) {
			return r.primary.ListAdmin(c, params)
		},
		func(c context.Context) (*domain.AdminContentPage, error) {
			return r.fallback.ListAdmin(c, params)
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
	conditions := []string{"c.archived_at IS NULL", "c.deleted_at IS NULL"}
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

	query := baseContentSelect + " WHERE c.short_id = $1 AND c.archived_at IS NULL AND c.deleted_at IS NULL"

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
WHERE c.short_id = $1 AND c.archived_at IS NULL AND c.deleted_at IS NULL
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

const baseAdminContentSelect = `
SELECT
    c.id::text,
    c.short_id,
    c.content_type,
    c.channel_id,
    c.disc_id::text,
    COALESCE(c.title, ''),
    COALESCE(c.description, ''),
    c.duration_seconds,
    COALESCE(c.is_360, false),
    COALESCE(c.tags, '{}'),
    c.status,
    c.document_key,
    c.document_mime_type,
    c.published_at,
    c.archived_at,
    c.created_at,
    c.updated_at
FROM contents c
`

func adminSortKeyExprAndDir(sort domain.ContentSort) (string, int, bool, error) {
	switch sort {
	case domain.ContentSortUpdatedDesc, "":
		return "date_trunc('milliseconds', c.updated_at)", -1, false, nil
	case domain.ContentSortUpdatedAsc:
		return "date_trunc('milliseconds', c.updated_at)", 1, false, nil
	case domain.ContentSortCreatedDesc:
		return "date_trunc('milliseconds', c.created_at)", -1, false, nil
	case domain.ContentSortCreatedAsc:
		return "date_trunc('milliseconds', c.created_at)", 1, false, nil
	case domain.ContentSortTitleAsc:
		return `c.title COLLATE "C"`, 1, true, nil
	case domain.ContentSortTitleDesc:
		return `c.title COLLATE "C"`, -1, true, nil
	default:
		return "", 0, false, fmt.Errorf("invalid sort: %q", sort)
	}
}

func buildAdminBaseConditions(params mongoRepo.AdminListParams) ([]string, []any, int) {
	// 論理削除済みのみ除外し、アーカイブ済み（archived_at IS NOT NULL）は管理一覧に含める。
	conditions := []string{"c.deleted_at IS NULL"}
	args := []any{}
	argIdx := 1

	if params.Status != nil {
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", argIdx))
		args = append(args, *params.Status)
		argIdx++
	}
	if params.ChannelID != nil {
		conditions = append(conditions, fmt.Sprintf("c.channel_id = $%d", argIdx))
		args = append(args, *params.ChannelID)
		argIdx++
	}
	return conditions, args, argIdx
}

func buildAdminRangeSQL(keyExpr string, dir int, isTitle bool, cur *mongoRepo.AdminCursor, isAfter bool, argIdx int) (string, []any, int) {
	var val any
	if isTitle {
		val = cur.Title
	} else {
		val = cur.Time.UTC()
	}

	var op, idOp string
	if isAfter {
		if dir == 1 {
			op = ">"
		} else {
			op = "<"
		}
		idOp = ">"
	} else {
		if dir == 1 {
			op = "<"
		} else {
			op = ">"
		}
		idOp = "<"
	}

	cond := fmt.Sprintf("(%s %s $%d OR (%s = $%d AND c.short_id COLLATE \"C\" %s $%d))",
		keyExpr, op, argIdx, keyExpr, argIdx, idOp, argIdx+1)
	return cond, []any{val, cur.ID}, argIdx + 2
}

// BuildAdminContentCountQuery constructs the SQL count query for admin contents list.
func BuildAdminContentCountQuery(params mongoRepo.AdminListParams) (string, []any) {
	conditions, args, _ := buildAdminBaseConditions(params)
	return "SELECT COUNT(*) FROM contents c WHERE " + strings.Join(conditions, " AND "), args
}

// BuildAdminContentListQuery constructs the SQL list query and arguments for admin contents cursor pagination.
func BuildAdminContentListQuery(params mongoRepo.AdminListParams) (string, []any, bool, error) {
	if params.After != nil && params.Before != nil {
		return "", nil, false, fmt.Errorf("%w: after and before cannot be specified together", mongoRepo.ErrInvalidCursor)
	}
	sort := params.Sort
	if sort == "" {
		sort = domain.ContentSortUpdatedDesc
	}
	keyExpr, dir, isTitle, err := adminSortKeyExprAndDir(sort)
	if err != nil {
		return "", nil, false, err
	}

	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	conditions, args, argIdx := buildAdminBaseConditions(params)
	reverse := false

	if params.Before != nil {
		if params.Before.Sort != sort {
			return "", nil, false, fmt.Errorf("%w: cursor sort mismatch", mongoRepo.ErrInvalidCursor)
		}
		reverse = true
		rangeCond, rangeArgs, nextIdx := buildAdminRangeSQL(keyExpr, dir, isTitle, params.Before, false, argIdx)
		conditions = append(conditions, rangeCond)
		args = append(args, rangeArgs...)
		argIdx = nextIdx
	} else if params.After != nil {
		if params.After.Sort != sort {
			return "", nil, false, fmt.Errorf("%w: cursor sort mismatch", mongoRepo.ErrInvalidCursor)
		}
		rangeCond, rangeArgs, nextIdx := buildAdminRangeSQL(keyExpr, dir, isTitle, params.After, true, argIdx)
		conditions = append(conditions, rangeCond)
		args = append(args, rangeArgs...)
		argIdx = nextIdx
	}

	var dirWord, idDirWord string
	if reverse {
		if dir == 1 {
			dirWord = "DESC"
		} else {
			dirWord = "ASC"
		}
		idDirWord = "DESC"
	} else {
		if dir == 1 {
			dirWord = "ASC"
		} else {
			dirWord = "DESC"
		}
		idDirWord = "ASC"
	}

	query := baseAdminContentSelect +
		" WHERE " + strings.Join(conditions, " AND ") +
		fmt.Sprintf(" ORDER BY %s %s, c.short_id COLLATE \"C\" %s LIMIT $%d", keyExpr, dirWord, idDirWord, argIdx)
	args = append(args, limit+1)
	return query, args, reverse, nil
}

// BuildAdminContentExistsQuery constructs an existence probe query strictly before or after a cursor.
func BuildAdminContentExistsQuery(params mongoRepo.AdminListParams, cur *mongoRepo.AdminCursor, isAfter bool) (string, []any, error) {
	sort := params.Sort
	if sort == "" {
		sort = domain.ContentSortUpdatedDesc
	}
	keyExpr, dir, isTitle, err := adminSortKeyExprAndDir(sort)
	if err != nil {
		return "", nil, err
	}
	conditions, args, argIdx := buildAdminBaseConditions(params)
	rangeCond, rangeArgs, _ := buildAdminRangeSQL(keyExpr, dir, isTitle, cur, isAfter, argIdx)
	conditions = append(conditions, rangeCond)
	args = append(args, rangeArgs...)
	return "SELECT EXISTS(SELECT 1 FROM contents c WHERE " + strings.Join(conditions, " AND ") + ")", args, nil
}

type adminPgQuerier interface {
	Count(ctx context.Context, query string, args ...any) (int, error)
	QueryRows(ctx context.Context, query string, args ...any) ([]*domain.Content, error)
	Exists(ctx context.Context, query string, args ...any) (bool, error)
}

type pgxPoolAdminQuerier struct {
	pool *pgxpool.Pool
}

func (q *pgxPoolAdminQuerier) Count(ctx context.Context, query string, args ...any) (int, error) {
	var total int
	if err := q.pool.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (q *pgxPoolAdminQuerier) QueryRows(ctx context.Context, query string, args ...any) ([]*domain.Content, error) {
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*domain.Content
	for rows.Next() {
		c, scanErr := scanAdminContent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (q *pgxPoolAdminQuerier) Exists(ctx context.Context, query string, args ...any) (bool, error) {
	var exists bool
	if err := q.pool.QueryRow(ctx, query, args...).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// ListAdmin executes fallback query for admin contents cursor pagination against PostgreSQL.
func (s *PostgresContentStore) ListAdmin(ctx context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
	if s.pool == nil {
		return nil, errors.New("postgres connection pool is nil")
	}

	querier := &pgxPoolAdminQuerier{pool: s.pool}
	return pgRepo.WithRetry(ctx, s.pool, func() (*domain.AdminContentPage, error) {
		return ListAdminWithQuerier(ctx, querier, params)
	})
}

// ListAdminWithQuerier assembles cursor-paginated admin pages using the provided query runner.
func ListAdminWithQuerier(ctx context.Context, querier adminPgQuerier, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
	sort := params.Sort
	if sort == "" {
		sort = domain.ContentSortUpdatedDesc
	}
	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	params.Sort = sort
	params.Limit = limit

	listQuery, listArgs, reverse, err := BuildAdminContentListQuery(params)
	if err != nil {
		return nil, err
	}
	countQuery, countArgs := BuildAdminContentCountQuery(params)

	total, err := querier.Count(ctx, countQuery, countArgs...)
	if err != nil {
		return nil, err
	}

	items, err := querier.QueryRows(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return &domain.AdminContentPage{
			Contents: []*domain.Content{},
			Total:    total,
			HasNext:  false,
			HasPrev:  false,
		}, nil
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	if reverse {
		slices.Reverse(items)
	}

	var hasNext, hasPrev bool
	if reverse {
		hasPrev = hasMore
		last := items[len(items)-1]
		existsQuery, existsArgs, err := BuildAdminContentExistsQuery(params, adminCursorFromContent(sort, last), true)
		if err != nil {
			return nil, err
		}
		hasNext, err = querier.Exists(ctx, existsQuery, existsArgs...)
		if err != nil {
			return nil, err
		}
	} else {
		hasNext = hasMore
		if params.After != nil {
			first := items[0]
			existsQuery, existsArgs, err := BuildAdminContentExistsQuery(params, adminCursorFromContent(sort, first), false)
			if err != nil {
				return nil, err
			}
			hasPrev, err = querier.Exists(ctx, existsQuery, existsArgs...)
			if err != nil {
				return nil, err
			}
		}
	}

	var nextCursor, prevCursor *string
	if hasNext {
		nc, err := mongoRepo.EncodeAdminCursor(sort, items[len(items)-1])
		if err != nil {
			return nil, err
		}
		nextCursor = &nc
	}
	if hasPrev {
		pc, err := mongoRepo.EncodeAdminCursor(sort, items[0])
		if err != nil {
			return nil, err
		}
		prevCursor = &pc
	}

	return &domain.AdminContentPage{
		Contents:   items,
		Total:      total,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}, nil
}

func adminCursorFromContent(sort domain.ContentSort, c *domain.Content) *mongoRepo.AdminCursor {
	cur := &mongoRepo.AdminCursor{
		Sort:  sort,
		Title: c.Title,
		ID:    c.ShortID,
	}
	if sort == domain.ContentSortCreatedDesc || sort == domain.ContentSortCreatedAsc {
		cur.Time = time.UnixMilli(c.CreatedAt.UnixMilli()).UTC()
	} else {
		cur.Time = time.UnixMilli(c.UpdatedAt.UnixMilli()).UTC()
	}
	return cur
}

func scanAdminContent(row rowScanner) (*domain.Content, error) {
	var c domain.Content
	err := row.Scan(
		&c.ID,
		&c.ShortID,
		&c.ContentType,
		&c.ChannelID,
		&c.DiscID,
		&c.Title,
		&c.Description,
		&c.DurationSeconds,
		&c.Is360,
		&c.Tags,
		&c.Status,
		&c.DocumentKey,
		&c.DocumentMimeType,
		&c.PublishedAt,
		&c.ArchivedAt,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if c.Tags == nil {
		c.Tags = []string{}
	}
	return &c, nil
}
