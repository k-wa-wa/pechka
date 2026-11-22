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
)

type mockPrimarySubtitleRepo struct {
	fn func(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error)
}

func (m *mockPrimarySubtitleRepo) GetByShortIDAndLanguage(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
	if m.fn != nil {
		return m.fn(ctx, shortID, language)
	}
	return nil, nil
}

type mockFallbackSubtitleStore struct {
	fn func(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error)
}

func (m *mockFallbackSubtitleStore) GetByShortIDAndLanguage(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
	if m.fn != nil {
		return m.fn(ctx, shortID, language)
	}
	return nil, nil
}

func TestBuildSubtitleQuery(t *testing.T) {
	q := fallback.BuildSubtitleQuery()
	if !strings.Contains(q, "t.status = 'published'") {
		t.Errorf("query missing published status check: %s", q)
	}
	if !strings.Contains(q, "c.archived_at IS NULL") {
		t.Errorf("query missing archived_at check: %s", q)
	}
	if !strings.Contains(q, "c.deleted_at IS NULL") {
		t.Errorf("query missing deleted_at check: %s", q)
	}
	if !strings.Contains(q, "ORDER BY cu.seq, cu.start_ms, cu.id") {
		t.Errorf("query missing cue sorting: %s", q)
	}
}

func TestFallbackSubtitleRepository_PrimarySuccess(t *testing.T) {
	primaryCalled := false
	fallbackCalled := false

	primary := &mockPrimarySubtitleRepo{
		fn: func(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
			primaryCalled = true
			return &domain.MongoSubtitle{ShortID: shortID, Language: language}, nil
		},
	}
	fbStore := &mockFallbackSubtitleStore{
		fn: func(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
			fallbackCalled = true
			return &domain.MongoSubtitle{ShortID: shortID + "_fb"}, nil
		},
	}

	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:    "mongo",
		Timeout: 100 * time.Millisecond,
	})
	repo := fallback.NewSubtitleRepositoryWithStore(primary, fbStore, cb)

	res, err := repo.GetByShortIDAndLanguage(context.Background(), "s1", "ja")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ShortID != "s1" {
		t.Errorf("expected s1, got %s", res.ShortID)
	}
	if !primaryCalled {
		t.Error("expected primary to be called")
	}
	if fallbackCalled {
		t.Error("fallback should not be called")
	}
}

func TestFallbackSubtitleRepository_NotFoundDoesNotFallback(t *testing.T) {
	fallbackCalled := false
	primary := &mockPrimarySubtitleRepo{
		fn: func(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
			return nil, mongo.ErrNoDocuments
		},
	}
	fbStore := &mockFallbackSubtitleStore{
		fn: func(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
			fallbackCalled = true
			return &domain.MongoSubtitle{ShortID: "fb"}, nil
		},
	}

	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:    "mongo",
		Timeout: 100 * time.Millisecond,
	})
	repo := fallback.NewSubtitleRepositoryWithStore(primary, fbStore, cb)

	res, err := repo.GetByShortIDAndLanguage(context.Background(), "s1", "ja")
	if !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("expected mongo.ErrNoDocuments, got %v", err)
	}
	if res != nil {
		t.Errorf("expected nil result, got %v", res)
	}
	if fallbackCalled {
		t.Error("fallback should not be called on ErrNoDocuments")
	}
}

func TestFallbackSubtitleRepository_FallbackOnFailure(t *testing.T) {
	primary := &mockPrimarySubtitleRepo{
		fn: func(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
			return nil, errors.New("mongo connection timeout")
		},
	}
	fbStore := &mockFallbackSubtitleStore{
		fn: func(ctx context.Context, shortID, language string) (*domain.MongoSubtitle, error) {
			return &domain.MongoSubtitle{
				ID:       shortID + ":" + language,
				ShortID:  shortID,
				Language: language,
				Status:   "published",
				Cues: []domain.MongoCue{
					{StartMs: 0, EndMs: 1000, Text: "こんにちは"},
				},
			}, nil
		},
	}

	cb := fallback.NewCircuitBreaker(fallback.CircuitBreakerSettings{
		Name:    "mongo",
		Timeout: 100 * time.Millisecond,
	})
	repo := fallback.NewSubtitleRepositoryWithStore(primary, fbStore, cb)

	res, err := repo.GetByShortIDAndLanguage(context.Background(), "s1", "ja")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ShortID != "s1" || len(res.Cues) != 1 || res.Cues[0].Text != "こんにちは" {
		t.Errorf("unexpected subtitle response: %v", res)
	}
}
