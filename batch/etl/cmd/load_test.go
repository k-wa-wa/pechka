package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("DISC_LABEL", "DISC_001")
	t.Setenv("CONTENT_TITLE", "Movie 1")
	t.Setenv("CONTENT_TYPE", "video")
	t.Setenv("CHANNEL_ID", "bluray")
	t.Setenv("IS_360", "true")

	cfg := loadConfigFromEnv()
	if cfg.DiscLabel != "DISC_001" {
		t.Errorf("expected DiscLabel DISC_001, got %q", cfg.DiscLabel)
	}
	if cfg.ContentTitle != "Movie 1" {
		t.Errorf("expected ContentTitle Movie 1, got %q", cfg.ContentTitle)
	}
	if cfg.ContentType != "video" {
		t.Errorf("expected ContentType video, got %q", cfg.ContentType)
	}
	if cfg.ChannelID != "bluray" {
		t.Errorf("expected ChannelID bluray, got %q", cfg.ChannelID)
	}
	if !cfg.Is360 {
		t.Errorf("expected Is360 true, got false")
	}
}

func TestLoadConfigFromEnv_EmptyChannelID(t *testing.T) {
	t.Setenv("DISC_LABEL", "DISC_001")
	t.Setenv("CONTENT_TITLE", "")
	t.Setenv("CHANNEL_ID", "")

	cfg := loadConfigFromEnv()
	if cfg.ContentTitle != "DISC_001" {
		t.Errorf("expected fallback ContentTitle DISC_001, got %q", cfg.ContentTitle)
	}
	if cfg.ChannelID != "" {
		t.Errorf("expected empty ChannelID when env not set, got %q", cfg.ChannelID)
	}
}

func TestUpsertContent_MissingChannelIDReturnsError(t *testing.T) {
	cfg := LoadConfig{
		DiscLabel:    "DISC_001",
		ContentTitle: "Movie 1",
		ContentType:  "video",
		ChannelID:    "",
	}
	_, _, err := upsertContent(context.Background(), nil, "short-1", nil, cfg)
	if err == nil {
		t.Fatal("expected error when ChannelID is empty, got nil")
	}
	if !strings.Contains(err.Error(), "CHANNEL_ID env var is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

type mockRow struct {
	scanFn func(dest ...any) error
}

func (r mockRow) Scan(dest ...any) error {
	return r.scanFn(dest...)
}

type mockDB struct {
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
	execFn     func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func (m *mockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return m.queryRowFn(ctx, sql, args...)
}

func (m *mockDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return m.execFn(ctx, sql, arguments...)
}

func TestLoadQueries_SoftDeleteGuards(t *testing.T) {
	if !strings.Contains(upsertContentQuery, "ON CONFLICT (disc_id) WHERE disc_id IS NOT NULL AND deleted_at IS NULL") {
		t.Errorf("upsertContentQuery must match partial unique index predicate: %s", upsertContentQuery)
	}
	if !strings.Contains(reuseContentQuery, "deleted_at IS NULL") {
		t.Errorf("reuseContentQuery must exclude deleted_at: %s", reuseContentQuery)
	}
	if !strings.Contains(markContentReadyQuery, "deleted_at IS NULL") {
		t.Errorf("markContentReadyQuery must exclude deleted_at: %s", markContentReadyQuery)
	}
	if !strings.Contains(getContentIDByShortIDQuery, "deleted_at IS NULL") {
		t.Errorf("getContentIDByShortIDQuery must exclude deleted_at: %s", getContentIDByShortIDQuery)
	}
}

func TestUpsertContent_Patterns(t *testing.T) {
	discID := "disc-1"
	cfg := LoadConfig{
		DiscLabel:    "DISC_001",
		ContentTitle: "Movie 1",
		ContentType:  "video",
		ChannelID:    "bluray",
	}

	// 1. Initial ingest / re-ingest after soft delete -> returns proposed short_id
	dbNew := &mockDB{
		queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*string)) = "uuid-new"
				*(dest[1].(*string)) = "proposed-short"
				return nil
			}}
		},
	}
	cid, sid, err := upsertContent(context.Background(), dbNew, "proposed-short", &discID, cfg)
	if err != nil || cid != "uuid-new" || sid != "proposed-short" {
		t.Fatalf("unexpected new ingest result: cid=%q sid=%q err=%v", cid, sid, err)
	}

	// 2. Existing undeleted row reused -> returns existing short_id
	dbReuse := &mockDB{
		queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*string)) = "uuid-existing"
				*(dest[1].(*string)) = "existing-short"
				return nil
			}}
		},
	}
	cid, sid, err = upsertContent(context.Background(), dbReuse, "proposed-short", &discID, cfg)
	if err != nil || cid != "uuid-existing" || sid != "existing-short" {
		t.Fatalf("unexpected reuse result: cid=%q sid=%q err=%v", cid, sid, err)
	}
}

func TestReuseGetAndMarkReady_DeletedReturnsError(t *testing.T) {
	dbNoRows := &mockDB{
		queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return mockRow{scanFn: func(_ ...any) error {
				return pgx.ErrNoRows
			}}
		},
		execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		},
	}

	if _, _, err := reuseContent(context.Background(), dbNoRows, "deleted-id"); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("reuseContent expected ErrNoRows for deleted content, got %v", err)
	}
	if _, err := getContentIDByShortID(context.Background(), dbNoRows, "deleted-short"); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("getContentIDByShortID expected ErrNoRows for deleted content, got %v", err)
	}
	if err := markContentReady(context.Background(), dbNoRows, "deleted-id"); err == nil || !strings.Contains(err.Error(), "not found or deleted") {
		t.Errorf("markContentReady expected explicit error when 0 rows affected, got %v", err)
	}

	dbUpdated := &mockDB{
		execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}
	if err := markContentReady(context.Background(), dbUpdated, "active-id"); err != nil {
		t.Errorf("markContentReady expected nil on 1 row affected, got %v", err)
	}
}
