package fallback_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	"github.com/k-wa-wa/pechka/api/internal/repository/fallback"
	mongoRepo "github.com/k-wa-wa/pechka/api/internal/repository/mongo"
)

type mockPrimaryContentRepo struct {
	listFn         func(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error)
	getByShortIDFn func(ctx context.Context, shortID string) (*domain.MongoContent, error)
	getVariantsFn  func(ctx context.Context, shortID string) ([]domain.MongoVariant, error)
	listAdminFn    func(ctx context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error)
}

func (m *mockPrimaryContentRepo) List(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return nil, nil
}

func (m *mockPrimaryContentRepo) GetByShortID(ctx context.Context, shortID string) (*domain.MongoContent, error) {
	if m.getByShortIDFn != nil {
		return m.getByShortIDFn(ctx, shortID)
	}
	return nil, nil
}

func (m *mockPrimaryContentRepo) GetVariantsByShortID(ctx context.Context, shortID string) ([]domain.MongoVariant, error) {
	if m.getVariantsFn != nil {
		return m.getVariantsFn(ctx, shortID)
	}
	return nil, nil
}

func (m *mockPrimaryContentRepo) ListAdmin(ctx context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
	if m.listAdminFn != nil {
		return m.listAdminFn(ctx, params)
	}
	return &domain.AdminContentPage{Contents: []*domain.Content{}}, nil
}

type mockFallbackContentStore struct {
	listFn         func(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error)
	getByShortIDFn func(ctx context.Context, shortID string) (*domain.MongoContent, error)
	getVariantsFn  func(ctx context.Context, shortID string) ([]domain.MongoVariant, error)
	listAdminFn    func(ctx context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error)
}

func (m *mockFallbackContentStore) List(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return nil, nil
}

func (m *mockFallbackContentStore) GetByShortID(ctx context.Context, shortID string) (*domain.MongoContent, error) {
	if m.getByShortIDFn != nil {
		return m.getByShortIDFn(ctx, shortID)
	}
	return nil, nil
}

func (m *mockFallbackContentStore) GetVariantsByShortID(ctx context.Context, shortID string) ([]domain.MongoVariant, error) {
	if m.getVariantsFn != nil {
		return m.getVariantsFn(ctx, shortID)
	}
	return nil, nil
}

func (m *mockFallbackContentStore) ListAdmin(ctx context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
	if m.listAdminFn != nil {
		return m.listAdminFn(ctx, params)
	}
	return &domain.AdminContentPage{Contents: []*domain.Content{}}, nil
}

func TestBuildContentListQuery(t *testing.T) {
	cType := domain.ContentTypeVideo
	chID := "ch-123"
	status := domain.ContentStatusReady

	query, args := fallback.BuildContentListQuery(mongoRepo.ListParams{
		ContentType: &cType,
		ChannelID:   &chID,
		Status:      &status,
		Limit:       50,
		Offset:      10,
	})

	if !strings.Contains(query, "c.archived_at IS NULL") {
		t.Errorf("query missing archived_at check: %s", query)
	}
	if !strings.Contains(query, "c.deleted_at IS NULL") {
		t.Errorf("query missing deleted_at check: %s", query)
	}
	if !strings.Contains(query, "c.content_type = $1") {
		t.Errorf("query missing content_type: %s", query)
	}
	if !strings.Contains(query, "c.channel_id = $2") {
		t.Errorf("query missing channel_id: %s", query)
	}
	if !strings.Contains(query, "c.status = $3") {
		t.Errorf("query missing status: %s", query)
	}
	if !strings.Contains(query, "LIMIT $4 OFFSET $5") {
		t.Errorf("query missing limit/offset: %s", query)
	}

	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d", len(args))
	}
	if args[0] != cType || args[1] != chID || args[2] != status || args[3] != int64(50) || args[4] != int64(10) {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestFallbackContentRepository_List_PrimarySuccess(t *testing.T) {
	primaryCalled := false
	fallbackCalled := false

	primary := &mockPrimaryContentRepo{
		listFn: func(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error) {
			primaryCalled = true
			return []*domain.MongoContent{{ID: "c1", Title: "Title 1"}}, nil
		},
	}
	fbStore := &mockFallbackContentStore{
		listFn: func(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error) {
			fallbackCalled = true
			return []*domain.MongoContent{{ID: "c1_fb"}}, nil
		},
	}

	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:    "mongo",
		Timeout: 100 * time.Millisecond,
	})
	repo := fallback.NewContentRepositoryWithStore(primary, fbStore, cb)

	res, err := repo.List(context.Background(), mongoRepo.ListParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 || res[0].ID != "c1" {
		t.Errorf("unexpected result: %v", res)
	}
	if !primaryCalled {
		t.Error("expected primary to be called")
	}
	if fallbackCalled {
		t.Error("fallback should not be called")
	}
}

func TestFallbackContentRepository_List_FallbackOnFailure(t *testing.T) {
	primary := &mockPrimaryContentRepo{
		listFn: func(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error) {
			return nil, errors.New("mongo cluster down")
		},
	}
	fbStore := &mockFallbackContentStore{
		listFn: func(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error) {
			return []*domain.MongoContent{{ID: "c1_from_pg", Title: "From Postgres"}}, nil
		},
	}

	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:    "mongo",
		Timeout: 100 * time.Millisecond,
	})
	repo := fallback.NewContentRepositoryWithStore(primary, fbStore, cb)

	res, err := repo.List(context.Background(), mongoRepo.ListParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 || res[0].ID != "c1_from_pg" {
		t.Errorf("expected fallback result, got %v", res)
	}
}

func TestFallbackContentRepository_GetByShortID_NotFoundDoesNotFallback(t *testing.T) {
	primary := &mockPrimaryContentRepo{
		getByShortIDFn: func(ctx context.Context, shortID string) (*domain.MongoContent, error) {
			return nil, mongo.ErrNoDocuments
		},
	}
	fallbackCalled := false
	fbStore := &mockFallbackContentStore{
		getByShortIDFn: func(ctx context.Context, shortID string) (*domain.MongoContent, error) {
			fallbackCalled = true
			return &domain.MongoContent{ID: "should_not_reach"}, nil
		},
	}

	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:    "mongo",
		Timeout: 100 * time.Millisecond,
	})
	repo := fallback.NewContentRepositoryWithStore(primary, fbStore, cb)

	res, err := repo.GetByShortID(context.Background(), "nonexistent")
	if !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("expected mongo.ErrNoDocuments, got %v", err)
	}
	if res != nil {
		t.Errorf("expected nil content, got %v", res)
	}
	if fallbackCalled {
		t.Error("fallback should not be called when primary returns mongo.ErrNoDocuments")
	}
	if cb.State() != fallback.StateClosed {
		t.Errorf("circuit breaker should remain closed, got %v", cb.State())
	}
}

func TestFallbackContentRepository_GetVariantsByShortID_Fallback(t *testing.T) {
	primary := &mockPrimaryContentRepo{
		getVariantsFn: func(ctx context.Context, shortID string) ([]domain.MongoVariant, error) {
			return nil, errors.New("timeout")
		},
	}
	fbStore := &mockFallbackContentStore{
		getVariantsFn: func(ctx context.Context, shortID string) ([]domain.MongoVariant, error) {
			return []domain.MongoVariant{{VariantType: "hls", HLSKey: "path/to/hls.m3u8"}}, nil
		},
	}

	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:    "mongo",
		Timeout: 100 * time.Millisecond,
	})
	repo := fallback.NewContentRepositoryWithStore(primary, fbStore, cb)

	res, err := repo.GetVariantsByShortID(context.Background(), "short-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 || res[0].VariantType != "hls" {
		t.Errorf("unexpected variants: %v", res)
	}
}

func TestBuildAdminContentListQuery(t *testing.T) {
	ts := time.UnixMilli(1700000000123).UTC()
	status := domain.ContentStatusReady
	ch := "tech-feed"

	cases := []struct {
		sort        domain.ContentSort
		wantOrderBy string
		wantAfterOp string
	}{
		{
			sort:        domain.ContentSortUpdatedDesc,
			wantOrderBy: `ORDER BY date_trunc('milliseconds', c.updated_at) DESC, c.short_id COLLATE "C" ASC`,
			wantAfterOp: `date_trunc('milliseconds', c.updated_at) < $`,
		},
		{
			sort:        domain.ContentSortUpdatedAsc,
			wantOrderBy: `ORDER BY date_trunc('milliseconds', c.updated_at) ASC, c.short_id COLLATE "C" ASC`,
			wantAfterOp: `date_trunc('milliseconds', c.updated_at) > $`,
		},
		{
			sort:        domain.ContentSortCreatedDesc,
			wantOrderBy: `ORDER BY date_trunc('milliseconds', c.created_at) DESC, c.short_id COLLATE "C" ASC`,
			wantAfterOp: `date_trunc('milliseconds', c.created_at) < $`,
		},
		{
			sort:        domain.ContentSortCreatedAsc,
			wantOrderBy: `ORDER BY date_trunc('milliseconds', c.created_at) ASC, c.short_id COLLATE "C" ASC`,
			wantAfterOp: `date_trunc('milliseconds', c.created_at) > $`,
		},
		{
			sort:        domain.ContentSortTitleAsc,
			wantOrderBy: `ORDER BY c.title COLLATE "C" ASC, c.short_id COLLATE "C" ASC`,
			wantAfterOp: `c.title COLLATE "C" > $`,
		},
		{
			sort:        domain.ContentSortTitleDesc,
			wantOrderBy: `ORDER BY c.title COLLATE "C" DESC, c.short_id COLLATE "C" ASC`,
			wantAfterOp: `c.title COLLATE "C" < $`,
		},
	}

	for _, tc := range cases {
		cur := &mongoRepo.AdminCursor{Sort: tc.sort, Time: ts, Title: "Alpha", ID: "s1"}

		// After query
		qAfter, argsAfter, revAfter, err := fallback.BuildAdminContentListQuery(mongoRepo.AdminListParams{
			Status:    &status,
			ChannelID: &ch,
			Sort:      tc.sort,
			After:     cur,
			Limit:     10,
		})
		if err != nil {
			t.Fatalf("%s after: %v", tc.sort, err)
		}
		if revAfter {
			t.Errorf("%s after should not be reverse", tc.sort)
		}
		if !strings.Contains(qAfter, "c.deleted_at IS NULL") {
			t.Errorf("%s missing c.deleted_at IS NULL: %s", tc.sort, qAfter)
		}
		if strings.Contains(qAfter, "c.archived_at IS NULL") {
			t.Errorf("%s admin list must NOT exclude archived_at: %s", tc.sort, qAfter)
		}
		if strings.Contains(qAfter, "OFFSET") {
			t.Errorf("%s must NOT use OFFSET: %s", tc.sort, qAfter)
		}
		if !strings.Contains(qAfter, tc.wantOrderBy) {
			t.Errorf("%s missing ORDER BY %q in %s", tc.sort, tc.wantOrderBy, qAfter)
		}
		if !strings.Contains(qAfter, tc.wantAfterOp) {
			t.Errorf("%s missing range condition %q in %s", tc.sort, tc.wantAfterOp, qAfter)
		}
		if len(argsAfter) != 5 || argsAfter[4] != 11 {
			t.Errorf("%s unexpected argsAfter: %v", tc.sort, argsAfter)
		}

		// Before query (reversed order)
		qBefore, _, revBefore, err := fallback.BuildAdminContentListQuery(mongoRepo.AdminListParams{
			Sort:   tc.sort,
			Before: cur,
			Limit:  10,
		})
		if err != nil {
			t.Fatalf("%s before: %v", tc.sort, err)
		}
		if !revBefore {
			t.Errorf("%s before should be reverse", tc.sort)
		}
		if !strings.Contains(qBefore, `c.short_id COLLATE "C" DESC`) {
			t.Errorf("%s before must reverse tiebreaker to DESC: %s", tc.sort, qBefore)
		}
	}

	// Count query
	qCount, countArgs := fallback.BuildAdminContentCountQuery(mongoRepo.AdminListParams{
		Status:    &status,
		ChannelID: &ch,
	})
	if !strings.Contains(qCount, "c.deleted_at IS NULL") || strings.Contains(qCount, "archived_at") || len(countArgs) != 2 {
		t.Errorf("unexpected count query %q args=%v", qCount, countArgs)
	}
}

func TestFallbackContentRepository_ListAdmin_PrimaryAndFallback(t *testing.T) {
	t.Run("primary success", func(t *testing.T) {
		primary := &mockPrimaryContentRepo{
			listAdminFn: func(_ context.Context, params mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
				if params.Sort != domain.ContentSortUpdatedDesc || params.Limit != 20 {
					t.Errorf("expected sanitized defaults, got %+v", params)
				}
				return &domain.AdminContentPage{
					Contents: []*domain.Content{{ID: "uuid-1", ShortID: "s1"}},
					Total:    1,
				}, nil
			},
		}
		fbStore := &mockFallbackContentStore{
			listAdminFn: func(_ context.Context, _ mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
				t.Fatal("fallback should not be called")
				return nil, nil
			},
		}

		cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{Name: "mongo", Timeout: 100 * time.Millisecond})
		repo := fallback.NewContentRepositoryWithStore(primary, fbStore, cb)

		page, err := repo.ListAdmin(context.Background(), mongoRepo.AdminListParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Contents) != 1 || page.Contents[0].ShortID != "s1" {
			t.Errorf("unexpected page: %+v", page)
		}
	})

	t.Run("fallback on ErrUnmigratedReadModel and mongo error", func(t *testing.T) {
		primary := &mockPrimaryContentRepo{
			listAdminFn: func(_ context.Context, _ mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
				return nil, mongoRepo.ErrUnmigratedReadModel
			},
		}
		fbStore := &mockFallbackContentStore{
			listAdminFn: func(_ context.Context, _ mongoRepo.AdminListParams) (*domain.AdminContentPage, error) {
				return &domain.AdminContentPage{
					Contents: []*domain.Content{{ID: "uuid-pg", ShortID: "pg-1"}},
					Total:    1,
				}, nil
			},
		}

		cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{Name: "mongo", Timeout: 100 * time.Millisecond})
		repo := fallback.NewContentRepositoryWithStore(primary, fbStore, cb)

		page, err := repo.ListAdmin(context.Background(), mongoRepo.AdminListParams{Sort: domain.ContentSortTitleAsc, Limit: 10})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Contents) != 1 || page.Contents[0].ShortID != "pg-1" {
			t.Errorf("expected postgres fallback page, got %+v", page)
		}
	})
}

type mockAdminPgQuerier struct {
	countFn     func(ctx context.Context, query string, args ...any) (int, error)
	queryRowsFn func(ctx context.Context, query string, args ...any) ([]*domain.Content, error)
	existsFn    func(ctx context.Context, query string, args ...any) (bool, error)
	existsCalls int
}

func (m *mockAdminPgQuerier) Count(ctx context.Context, query string, args ...any) (int, error) {
	if m.countFn != nil {
		return m.countFn(ctx, query, args...)
	}
	return 0, nil
}

func (m *mockAdminPgQuerier) QueryRows(ctx context.Context, query string, args ...any) ([]*domain.Content, error) {
	if m.queryRowsFn != nil {
		return m.queryRowsFn(ctx, query, args...)
	}
	return nil, nil
}

func (m *mockAdminPgQuerier) Exists(ctx context.Context, query string, args ...any) (bool, error) {
	m.existsCalls++
	if m.existsFn != nil {
		return m.existsFn(ctx, query, args...)
	}
	return false, nil
}

func TestListAdminWithQuerier_PageAssemblyAndBoundaries(t *testing.T) {
	ctx := context.Background()
	tsSameMs1 := time.Date(2025, 5, 1, 10, 0, 0, 123456000, time.UTC)
	tsSameMs2 := time.Date(2025, 5, 1, 10, 0, 0, 123789000, time.UTC)

	t.Run("0 items with cursor skips exists probe", func(t *testing.T) {
		q := &mockAdminPgQuerier{
			countFn: func(_ context.Context, _ string, _ ...any) (int, error) { return 5, nil },
			queryRowsFn: func(_ context.Context, _ string, _ ...any) ([]*domain.Content, error) {
				return []*domain.Content{}, nil
			},
		}
		cur := &mongoRepo.AdminCursor{Sort: domain.ContentSortUpdatedDesc, Time: tsSameMs1, ID: "s1"}
		page, err := fallback.ListAdminWithQuerier(ctx, q, mongoRepo.AdminListParams{Sort: domain.ContentSortUpdatedDesc, After: cur, Limit: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Contents) != 0 || page.Total != 5 || page.HasNext || page.HasPrev || page.NextCursor != nil || page.PrevCursor != nil {
			t.Errorf("unexpected empty page: %+v", page)
		}
		if q.existsCalls != 0 {
			t.Errorf("expected 0 exists calls on empty page, got %d", q.existsCalls)
		}
	})

	t.Run("exact limit items hasNext is false", func(t *testing.T) {
		items := []*domain.Content{
			{ID: "u1", ShortID: "s1", Title: "Same", UpdatedAt: tsSameMs1},
			{ID: "u2", ShortID: "s2", Title: "Same", UpdatedAt: tsSameMs2},
		}
		q := &mockAdminPgQuerier{
			countFn:     func(_ context.Context, _ string, _ ...any) (int, error) { return 2, nil },
			queryRowsFn: func(_ context.Context, _ string, _ ...any) ([]*domain.Content, error) { return items, nil },
		}
		page, err := fallback.ListAdminWithQuerier(ctx, q, mongoRepo.AdminListParams{Sort: domain.ContentSortUpdatedDesc, Limit: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Contents) != 2 || page.HasNext || page.HasPrev || page.NextCursor != nil || page.PrevCursor != nil {
			t.Errorf("unexpected exact-limit page: %+v", page)
		}
	})

	t.Run("limit+1 items with after truncates and verifies millisecond tiebreaker cursors", func(t *testing.T) {
		items := []*domain.Content{
			{ID: "u1", ShortID: "s1", Title: "Same", UpdatedAt: tsSameMs1},
			{ID: "u2", ShortID: "s2", Title: "Same", UpdatedAt: tsSameMs2},
			{ID: "u3", ShortID: "s3", Title: "Same", UpdatedAt: tsSameMs1},
		}
		var gotExistsQuery string
		var gotExistsArgs []any
		q := &mockAdminPgQuerier{
			countFn:     func(_ context.Context, _ string, _ ...any) (int, error) { return 4, nil },
			queryRowsFn: func(_ context.Context, _ string, _ ...any) ([]*domain.Content, error) { return items, nil },
			existsFn: func(_ context.Context, query string, args ...any) (bool, error) {
				gotExistsQuery = query
				gotExistsArgs = args
				return true, nil
			},
		}
		afterCur := &mongoRepo.AdminCursor{Sort: domain.ContentSortUpdatedDesc, Time: tsSameMs1.Truncate(time.Millisecond), ID: "s0"}
		page, err := fallback.ListAdminWithQuerier(ctx, q, mongoRepo.AdminListParams{Sort: domain.ContentSortUpdatedDesc, After: afterCur, Limit: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Contents) != 2 || page.Contents[0].ShortID != "s1" || page.Contents[1].ShortID != "s2" {
			t.Fatalf("unexpected truncated contents: %+v", page.Contents)
		}
		if !page.HasNext || !page.HasPrev || page.NextCursor == nil || page.PrevCursor == nil {
			t.Fatalf("expected cursors and flags true, got %+v", page)
		}
		nc, _ := mongoRepo.DecodeAdminCursor(*page.NextCursor, domain.ContentSortUpdatedDesc)
		pc, _ := mongoRepo.DecodeAdminCursor(*page.PrevCursor, domain.ContentSortUpdatedDesc)
		if nc.ID != "s2" || pc.ID != "s1" {
			t.Errorf("unexpected cursors: prev=%s next=%s", pc.ID, nc.ID)
		}
		if !strings.Contains(gotExistsQuery, `c.short_id COLLATE "C" < $`) || len(gotExistsArgs) != 2 || gotExistsArgs[1] != "s1" {
			t.Errorf("unexpected prev exists query/args: %s %v", gotExistsQuery, gotExistsArgs)
		}
	})

	t.Run("before cursor reverses slice and probes hasNext", func(t *testing.T) {
		reverseRows := []*domain.Content{
			{ID: "u3", ShortID: "s3", Title: "B", CreatedAt: tsSameMs1},
			{ID: "u2", ShortID: "s2", Title: "B", CreatedAt: tsSameMs2},
			{ID: "u1", ShortID: "s1", Title: "A", CreatedAt: tsSameMs1},
		}
		var gotExistsQuery string
		var gotExistsArgs []any
		q := &mockAdminPgQuerier{
			countFn:     func(_ context.Context, _ string, _ ...any) (int, error) { return 4, nil },
			queryRowsFn: func(_ context.Context, _ string, _ ...any) ([]*domain.Content, error) { return reverseRows, nil },
			existsFn: func(_ context.Context, query string, args ...any) (bool, error) {
				gotExistsQuery = query
				gotExistsArgs = args
				return true, nil
			},
		}
		beforeCur := &mongoRepo.AdminCursor{Sort: domain.ContentSortTitleAsc, Title: "C", ID: "s4"}
		page, err := fallback.ListAdminWithQuerier(ctx, q, mongoRepo.AdminListParams{Sort: domain.ContentSortTitleAsc, Before: beforeCur, Limit: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Contents) != 2 || page.Contents[0].ShortID != "s2" || page.Contents[1].ShortID != "s3" {
			t.Fatalf("expected reversed forward order [s2, s3], got %+v", page.Contents)
		}
		if !page.HasPrev || !page.HasNext || page.PrevCursor == nil || page.NextCursor == nil {
			t.Fatalf("expected hasPrev & hasNext true, got %+v", page)
		}
		if !strings.Contains(gotExistsQuery, `c.short_id COLLATE "C" > $`) || len(gotExistsArgs) != 2 || gotExistsArgs[1] != "s3" {
			t.Errorf("unexpected next exists query/args: %s %v", gotExistsQuery, gotExistsArgs)
		}
	})

	t.Run("exists error propagates", func(t *testing.T) {
		q := &mockAdminPgQuerier{
			countFn: func(_ context.Context, _ string, _ ...any) (int, error) { return 2, nil },
			queryRowsFn: func(_ context.Context, _ string, _ ...any) ([]*domain.Content, error) {
				return []*domain.Content{{ID: "u1", ShortID: "s1", UpdatedAt: tsSameMs1}}, nil
			},
			existsFn: func(_ context.Context, _ string, _ ...any) (bool, error) {
				return false, errors.New("pg exists failed")
			},
		}
		afterCur := &mongoRepo.AdminCursor{Sort: domain.ContentSortUpdatedDesc, Time: tsSameMs1, ID: "s0"}
		_, err := fallback.ListAdminWithQuerier(ctx, q, mongoRepo.AdminListParams{Sort: domain.ContentSortUpdatedDesc, After: afterCur, Limit: 2})
		if err == nil || !strings.Contains(err.Error(), "pg exists failed") {
			t.Fatalf("expected exists error to propagate, got %v", err)
		}
	})
}
