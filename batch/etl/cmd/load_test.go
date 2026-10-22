package cmd

import (
	"context"
	"strings"
	"testing"
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
