package main

import (
	"testing"
	"time"
)

func TestBuildPlaylist(t *testing.T) {
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

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && searchSubstr(s, substr))
}

func searchSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
