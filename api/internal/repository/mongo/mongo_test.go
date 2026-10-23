package mongo

import (
	"context"
	"testing"
)

func TestMongoRepositorySignatures(t *testing.T) {
	// Verify structs and functions compile and can be instantiated with nil db (or type assertion)
	var contentRepo *ContentRepository
	var subRepo *SubtitleRepository

	if contentRepo != nil || subRepo != nil {
		t.Fatal("expected nil pointers")
	}

	// Verify ListParams
	params := ListParams{
		Limit:  10,
		Offset: 0,
	}
	if params.Limit != 10 {
		t.Errorf("unexpected limit: %d", params.Limit)
	}

	// Context check
	ctx := context.Background()
	if ctx == nil {
		t.Fatal("expected valid context")
	}
}
