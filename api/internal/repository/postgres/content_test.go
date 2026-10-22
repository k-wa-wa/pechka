package postgres

import (
	"context"
	"strings"
	"testing"

	"github.com/k-wa-wa/pechka/api/internal/domain"
)

func TestCreate_MissingChannelIDReturnsError(t *testing.T) {
	repo := NewContentRepository(nil)
	_, err := repo.Create(context.Background(), CreateContentParams{
		ShortID:     "short-1",
		ContentType: domain.ContentTypeVideo,
		ChannelID:   "   ",
		Title:       "Test",
		Status:      domain.ContentStatusPending,
	})
	if err == nil {
		t.Fatal("expected error when ChannelID is empty, got nil")
	}
	if !strings.Contains(err.Error(), "channel_id is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBuildListContentsWhere(t *testing.T) {
	statusReady := domain.ContentStatusReady
	channelTechFeed := "tech-feed"

	where, args, nextIdx := buildListContentsWhere(ListContentsParams{
		Status:          &statusReady,
		ChannelID:       &channelTechFeed,
		IncludeArchived: false,
	})

	wantWhere := " WHERE status = $1 AND channel_id = $2 AND archived_at IS NULL"
	if where != wantWhere {
		t.Errorf("got where %q, want %q", where, wantWhere)
	}
	if len(args) != 2 || args[0] != statusReady || args[1] != channelTechFeed {
		t.Errorf("unexpected args: %v", args)
	}
	if nextIdx != 3 {
		t.Errorf("got nextIdx %d, want 3", nextIdx)
	}
}
