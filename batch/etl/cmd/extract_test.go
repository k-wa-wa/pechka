package cmd

import (
	"strings"
	"testing"
)

func TestResolveDiscKey(t *testing.T) {
	// 1. Regular disc label should be preserved as-is
	label := "MY_MOVIE_DISC"
	key := resolveDiscKey("/dev/null", label)
	if key != label {
		t.Errorf("expected %q, got %q", label, key)
	}

	// 2. DVD_VIDEO_RECORDER with non-existent or invalid device should fallback to timestamp suffix
	recLabel := "DVD_VIDEO_RECORDER"
	fallbackKey := resolveDiscKey("/dev/nonexistent_device_test", recLabel)
	if !strings.HasPrefix(fallbackKey, "DVD_VIDEO_RECORDER_") {
		t.Errorf("expected prefix DVD_VIDEO_RECORDER_, got %q", fallbackKey)
	}
	if fallbackKey == recLabel {
		t.Errorf("expected fallbackKey to differ from bare label %q", recLabel)
	}
}
