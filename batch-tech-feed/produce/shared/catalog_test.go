package shared_test

import (
	"strings"
	"testing"

	"github.com/k-wa-wa/pechka/batch-tech-feed/produce/shared"
)

func TestShortIDFor(t *testing.T) {
	sourceKey := "tech-feed:2026-08-02"

	id1 := shared.ShortIDFor(sourceKey)
	id2 := shared.ShortIDFor(sourceKey)

	t.Logf("Generated ID 1: %s", id1)
	t.Logf("Generated ID 2: %s", id2)

	if !strings.HasPrefix(id1, "tech-feed-2026-08-02-") {
		t.Errorf("expected prefix tech-feed-2026-08-02-, got %s", id1)
	}

	if id1 == id2 {
		t.Errorf("expected random suffix to be different, got identical: %s", id1)
	}

	if len(id1) > shared.ShortIDMax {
		t.Errorf("expected length <= %d, got %d", shared.ShortIDMax, len(id1))
	}
}
