package cmd

import (
	"context"
	"strings"
	"testing"
)

func TestRunTransform_MissingArgs(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		args []string
	}{
		{"empty args", []string{}},
		{"missing input", []string{"-output=/tmp", "-mode=720p", "-short-id=s1"}},
		{"missing output", []string{"-input=s3://bucket/test.mkv", "-mode=720p", "-short-id=s1"}},
		{"missing mode", []string{"-input=s3://bucket/test.mkv", "-output=/tmp", "-short-id=s1"}},
		{"missing short-id", []string{"-input=s3://bucket/test.mkv", "-output=/tmp", "-mode=720p"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunTransform(ctx, tt.args)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.name)
			}
			if !strings.Contains(err.Error(), "missing required arguments") {
				t.Errorf("expected missing required arguments error, got %v", err)
			}
		})
	}
}

func TestTranscodeVideoVariant_UnsupportedResolution(t *testing.T) {
	err := transcodeVideoVariant("dummy.mkv", t.TempDir(), "240p")
	if err == nil {
		t.Fatal("expected error for unsupported resolution, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported resolution: 240p") {
		t.Errorf("unexpected error message: %v", err)
	}
}
