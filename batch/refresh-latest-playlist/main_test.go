package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/minio/minio-go/v7"
)

func TestBuildPlaylist(t *testing.T) {
	now := time.Now()
	t.Run("single item", func(t *testing.T) {
		rows := []contentRow{
			{
				shortID:     "abc1",
				title:       "Video 1",
				publishedAt: &now,
			},
		}
		pl := buildPlaylist(rows)
		if pl == "" {
			t.Fatal("expected non-empty playlist")
		}
		if expected := "#EXTINF:-1,Video 1\n"; !contains(pl, expected) {
			t.Errorf("expected playlist to contain %s, got: %s", expected, pl)
		}
		if expected := "resources/hls/abc1/master.m3u8\n"; !contains(pl, expected) {
			t.Errorf("expected playlist to contain %s, got: %s", expected, pl)
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		pl := buildPlaylist(nil)
		expected := "#EXTM3U\n"
		if pl != expected {
			t.Errorf("expected %q, got %q", expected, pl)
		}
	})

	t.Run("multiple items and special characters", func(t *testing.T) {
		t1 := time.Now()
		t2 := t1.Add(-1 * time.Hour)
		rows := []contentRow{
			{
				shortID:     "id-001",
				title:       "タイトル: 日本語 / 特殊文字 & < > \" '",
				publishedAt: &t1,
			},
			{
				shortID:     "id-002",
				title:       "Commas, and quotes \"here\"",
				publishedAt: &t2,
			},
		}
		pl := buildPlaylist(rows)
		expectedStart := "#EXTM3U\n"
		if !strings.HasPrefix(pl, expectedStart) {
			t.Errorf("expected playlist to start with %q, got: %s", expectedStart, pl)
		}
		if !contains(pl, "#EXTINF:-1,タイトル: 日本語 / 特殊文字 & < > \" '\nresources/hls/id-001/master.m3u8\n") {
			t.Errorf("playlist missing first item, got: %s", pl)
		}
		if !contains(pl, "#EXTINF:-1,Commas, and quotes \"here\"\nresources/hls/id-002/master.m3u8\n") {
			t.Errorf("playlist missing second item, got: %s", pl)
		}
	})

	t.Run("title with embedded newlines", func(t *testing.T) {
		t1 := time.Now()
		rows := []contentRow{
			{
				shortID:     "id-newline",
				title:       "Line 1\nLine 2",
				publishedAt: &t1,
			},
		}
		pl := buildPlaylist(rows)
		// Currently buildPlaylist formats title directly with %s, so newlines are preserved across lines
		expected := "#EXTM3U\n#EXTINF:-1,Line 1\nLine 2\nresources/hls/id-newline/master.m3u8\n"
		if pl != expected {
			t.Errorf("expected %q, got %q", expected, pl)
		}
	})
}

func TestGetenv(t *testing.T) {
	t.Setenv("TEST_ENV_VAR", "set_value")
	if v := getenv("TEST_ENV_VAR", "default"); v != "set_value" {
		t.Errorf("expected set_value, got %s", v)
	}
	if v := getenv("NON_EXISTENT_VAR", "default"); v != "default" {
		t.Errorf("expected default, got %s", v)
	}
}

func TestPostgresDSN(t *testing.T) {
	t.Run("POSTGRES_DSN direct override", func(t *testing.T) {
		t.Setenv("POSTGRES_DSN", "postgres://user:pass@remote:5432/dbname?sslmode=verify-full")
		dsN := postgresDSN()
		if dsN != "postgres://user:pass@remote:5432/dbname?sslmode=verify-full" {
			t.Errorf("unexpected DSN: %s", dsN)
		}
	})

	t.Run("Individual DB_* environment variables", func(t *testing.T) {
		t.Setenv("POSTGRES_DSN", "")
		t.Setenv("DB_HOST", "db.internal")
		t.Setenv("DB_PORT", "5433")
		t.Setenv("DB_USER", "appuser")
		t.Setenv("DB_PASSWORD", "secret")
		t.Setenv("DB_NAME", "pechkadb")
		t.Setenv("SSL_MODE", "require")

		dsN := postgresDSN()
		expected := "host=db.internal port=5433 user=appuser password=secret dbname=pechkadb sslmode=require"
		if dsN != expected {
			t.Errorf("expected %q, got %q", expected, dsN)
		}
	})

	t.Run("Default values for DB_PORT and SSL_MODE", func(t *testing.T) {
		t.Setenv("POSTGRES_DSN", "")
		t.Setenv("DB_HOST", "localhost")
		t.Setenv("DB_PORT", "")
		t.Setenv("DB_USER", "postgres")
		t.Setenv("DB_PASSWORD", "pass")
		t.Setenv("DB_NAME", "defaultdb")
		t.Setenv("SSL_MODE", "")

		dsN := postgresDSN()
		expected := "host=localhost port=5432 user=postgres password=pass dbname=defaultdb sslmode=disable"
		if dsN != expected {
			t.Errorf("expected %q, got %q", expected, dsN)
		}
	})
}

func TestMustGetenv(t *testing.T) {
	t.Run("env var exists", func(t *testing.T) {
		t.Setenv("TEST_MUST_KEY", "hello")
		if v := mustGetenv("TEST_MUST_KEY"); v != "hello" {
			t.Errorf("expected hello, got %s", v)
		}
	})

	t.Run("env var missing causes exit", func(t *testing.T) {
		if os.Getenv("BE_CRASHER") == "1" {
			mustGetenv("DEFINITELY_MISSING_VAR_FOR_CRASH_TEST")
			return
		}
		cmd := exec.Command(os.Args[0], "-test.run=TestMustGetenv/env_var_missing_causes_exit")
		cmd.Env = append(os.Environ(), "BE_CRASHER=1")
		err := cmd.Run()
		if e, ok := err.(*exec.ExitError); ok && !e.Success() {
			return
		}
		t.Fatalf("process ran with err %v, want exit status 1", err)
	})
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

type mockPlaylistRows struct {
	items []contentRow
	idx   int
}

func (r *mockPlaylistRows) Next() bool {
	if r.idx < len(r.items) {
		r.idx++
		return true
	}
	return false
}

func (r *mockPlaylistRows) Scan(dest ...any) error {
	item := r.items[r.idx-1]
	*(dest[0].(*string)) = item.shortID
	*(dest[1].(*string)) = item.title
	*(dest[2].(**time.Time)) = item.publishedAt
	return nil
}

func (r *mockPlaylistRows) Err() error { return nil }
func (r *mockPlaylistRows) Close()     {}

type mockPlaylistTx struct {
	events *[]string
	items  []contentRow
}

func (tx *mockPlaylistTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "pg_advisory_xact_lock") {
		if len(args) != 1 || args[0] != latestPlaylistLockKey {
			return pgconn.CommandTag{}, errors.New("unexpected lock key")
		}
		*tx.events = append(*tx.events, "lock")
	}
	return pgconn.CommandTag{}, nil
}

func (tx *mockPlaylistTx) Query(_ context.Context, sql string, args ...any) (playlistRows, error) {
	if sql != latestPlaylistQuery {
		return nil, errors.New("unexpected query")
	}
	if len(args) != 1 || args[0] != latestPlaylistLimit {
		return nil, errors.New("unexpected limit")
	}
	*tx.events = append(*tx.events, "query")
	return &mockPlaylistRows{items: tx.items}, nil
}

func (tx *mockPlaylistTx) Commit(_ context.Context) error {
	*tx.events = append(*tx.events, "commit")
	return nil
}

func (tx *mockPlaylistTx) Rollback(_ context.Context) error {
	*tx.events = append(*tx.events, "rollback")
	return nil
}

type mockBeginner struct {
	tx      *mockPlaylistTx
	gotOpts pgx.TxOptions
	events  *[]string
}

func (b *mockBeginner) BeginTx(_ context.Context, txOptions pgx.TxOptions) (playlistTx, error) {
	b.gotOpts = txOptions
	*b.events = append(*b.events, "begin")
	return b.tx, nil
}

type mockUploader struct {
	events   *[]string
	uploaded string
	err      error
}

func (u *mockUploader) PutObject(_ context.Context, bucketName, objectName string, reader io.Reader, _ int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	*u.events = append(*u.events, "put")
	if u.err != nil {
		return minio.UploadInfo{}, u.err
	}
	data, _ := io.ReadAll(reader)
	u.uploaded = string(data)
	if bucketName != "test-bucket" || objectName != latestPlaylistKey || opts.ContentType != "application/vnd.apple.mpegurl" {
		return minio.UploadInfo{}, errors.New("unexpected PutObject arguments")
	}
	return minio.UploadInfo{}, nil
}

func TestRefreshPlaylist_LockOrderAndNullPublishedAt(t *testing.T) {
	if latestPlaylistLockKey != 0x706563686b610001 {
		t.Fatalf("unexpected lock key: %x", latestPlaylistLockKey)
	}
	if !strings.Contains(latestPlaylistQuery, "deleted_at IS NULL") ||
		!strings.Contains(latestPlaylistQuery, "archived_at IS NULL") ||
		!strings.Contains(latestPlaylistQuery, "ORDER BY published_at DESC NULLS LAST, updated_at DESC, short_id ASC") {
		t.Fatalf("unexpected latestPlaylistQuery: %s", latestPlaylistQuery)
	}

	now := time.Now().UTC()
	events := []string{}
	beginner := &mockBeginner{
		events: &events,
		tx: &mockPlaylistTx{
			events: &events,
			items: []contentRow{
				{shortID: "s1", title: "With PublishedAt", publishedAt: &now},
				{shortID: "s2", title: "Null PublishedAt", publishedAt: nil},
			},
		},
	}
	uploader := &mockUploader{events: &events}

	count, err := refreshPlaylist(context.Background(), beginner, uploader, "test-bucket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
	if beginner.gotOpts.IsoLevel != pgx.ReadCommitted {
		t.Errorf("expected ReadCommitted isolation, got %v", beginner.gotOpts.IsoLevel)
	}
	wantOrder := "begin,lock,query,put,commit,rollback"
	if got := strings.Join(events, ","); got != wantOrder {
		t.Errorf("execution order = %q, want %q", got, wantOrder)
	}
	wantBody := "#EXTM3U\n#EXTINF:-1,With PublishedAt\nresources/hls/s1/master.m3u8\n#EXTINF:-1,Null PublishedAt\nresources/hls/s2/master.m3u8\n"
	if uploader.uploaded != wantBody {
		t.Errorf("uploaded playlist = %q, want %q", uploader.uploaded, wantBody)
	}

	// PutObject failure rolls back without commit
	failEvents := []string{}
	failBeginner := &mockBeginner{
		events: &failEvents,
		tx: &mockPlaylistTx{
			events: &failEvents,
			items:  []contentRow{{shortID: "s1", title: "V1"}},
		},
	}
	failUploader := &mockUploader{events: &failEvents, err: errors.New("upload failed")}
	if _, err := refreshPlaylist(context.Background(), failBeginner, failUploader, "test-bucket"); err == nil {
		t.Fatal("expected error on upload failure")
	}
	if got := strings.Join(failEvents, ","); got != "begin,lock,query,put,rollback" {
		t.Errorf("fail execution order = %q", got)
	}
}
