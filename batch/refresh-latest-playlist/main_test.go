package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestBuildPlaylist(t *testing.T) {
	t.Run("single item", func(t *testing.T) {
		rows := []contentRow{
			{
				shortID:     "abc1",
				title:       "Video 1",
				publishedAt: time.Now(),
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
		rows := []contentRow{
			{
				shortID:     "id-001",
				title:       "タイトル: 日本語 / 特殊文字 & < > \" '",
				publishedAt: time.Now(),
			},
			{
				shortID:     "id-002",
				title:       "Commas, and quotes \"here\"",
				publishedAt: time.Now().Add(-1 * time.Hour),
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
		rows := []contentRow{
			{
				shortID:     "id-newline",
				title:       "Line 1\nLine 2",
				publishedAt: time.Now(),
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
