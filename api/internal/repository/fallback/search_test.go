package fallback_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	elasticRepo "github.com/k-wa-wa/pechka/api/internal/repository/elastic"
	"github.com/k-wa-wa/pechka/api/internal/repository/fallback"
)

type mockPrimarySearchRepo struct {
	fn func(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error)
}

func (m *mockPrimarySearchRepo) Search(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error) {
	if m.fn != nil {
		return m.fn(ctx, query, limit, offset)
	}
	return nil, nil
}

type mockFallbackSearchStore struct {
	fn func(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error)
}

func (m *mockFallbackSearchStore) Search(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error) {
	if m.fn != nil {
		return m.fn(ctx, query, limit, offset)
	}
	return nil, nil
}

func TestBuildSearchQuery(t *testing.T) {
	sql, args := fallback.BuildSearchQuery("test query", 30, 5)

	if !strings.Contains(sql, "c.archived_at IS NULL") {
		t.Errorf("query missing archived_at check: %s", sql)
	}
	if !strings.Contains(sql, "c.title ILIKE $1") {
		t.Errorf("query missing title ILIKE: %s", sql)
	}
	if !strings.Contains(sql, "c.description ILIKE $1") {
		t.Errorf("query missing description ILIKE: %s", sql)
	}
	if !strings.Contains(sql, "unnest(c.tags)") {
		t.Errorf("query missing tags unnest check: %s", sql)
	}
	if !strings.Contains(sql, "LIMIT $2 OFFSET $3") {
		t.Errorf("query missing LIMIT/OFFSET: %s", sql)
	}

	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(args))
	}
	if args[0] != "%test query%" || args[1] != 30 || args[2] != 5 {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestFallbackSearchRepository_PrimarySuccess(t *testing.T) {
	primaryCalled := false
	fallbackCalled := false

	primary := &mockPrimarySearchRepo{
		fn: func(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error) {
			primaryCalled = true
			return []*elasticRepo.SearchResult{
				{ShortID: "s1", Title: "Elastic Result"},
			}, nil
		},
	}
	fbStore := &mockFallbackSearchStore{
		fn: func(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error) {
			fallbackCalled = true
			return []*elasticRepo.SearchResult{
				{ShortID: "s1_fb", Title: "PG Result"},
			}, nil
		},
	}

	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:    "elastic",
		Timeout: 100 * time.Millisecond,
	})
	repo := fallback.NewSearchRepositoryWithStore(primary, fbStore, cb)

	results, err := repo.Search(context.Background(), "query", 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Title != "Elastic Result" {
		t.Errorf("unexpected result: %v", results)
	}
	if !primaryCalled {
		t.Error("expected primary to be called")
	}
	if fallbackCalled {
		t.Error("fallback should not be called on primary success")
	}
}

func TestFallbackSearchRepository_FallbackOnFailure(t *testing.T) {
	primary := &mockPrimarySearchRepo{
		fn: func(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error) {
			return nil, errors.New("es cluster down")
		},
	}
	fbStore := &mockFallbackSearchStore{
		fn: func(ctx context.Context, query string, limit, offset int) ([]*elasticRepo.SearchResult, error) {
			return []*elasticRepo.SearchResult{
				{
					ShortID:     "s1",
					Title:       "Postgres Found",
					ContentType: domain.ContentTypeVideo,
					Status:      domain.ContentStatusReady,
					Tags:        []string{"go"},
				},
			}, nil
		},
	}

	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:    "elastic",
		Timeout: 100 * time.Millisecond,
	})
	repo := fallback.NewSearchRepositoryWithStore(primary, fbStore, cb)

	results, err := repo.Search(context.Background(), "query", 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Title != "Postgres Found" {
		t.Errorf("unexpected result: %v", results)
	}
}
