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

type mockFallbackContentStore struct {
	listFn         func(ctx context.Context, params mongoRepo.ListParams) ([]*domain.MongoContent, error)
	getByShortIDFn func(ctx context.Context, shortID string) (*domain.MongoContent, error)
	getVariantsFn  func(ctx context.Context, shortID string) ([]domain.MongoVariant, error)
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
