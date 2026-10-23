package cmd

import (
	"context"
	"testing"
)

func TestProduceCmdArgs(t *testing.T) {
	ctx := context.Background()
	// Test calling render with insufficient args returns error cleanly
	err := RunRender(ctx, []string{})
	if err == nil {
		t.Error("expected error for empty args, got nil")
	}

	// Test calling publish with insufficient args returns error cleanly
	err = RunPublish(ctx, []string{})
	if err == nil {
		t.Error("expected error for empty args, got nil")
	}
}
