package main

import (
	"testing"
)

func TestCurrentNamespace(t *testing.T) {
	// Without the kubernetes serviceaccount namespace file mounted,
	// currentNamespace should fallback to default "pechka".
	ns := currentNamespace()
	if ns != "pechka" {
		t.Errorf("expected default namespace 'pechka', got %s", ns)
	}
}

func TestResolveNodeID(t *testing.T) {
	// Fallback test
	t.Setenv("NODE_ID", "")
	t.Setenv("POD_NAME", "")
	if id, err := resolveNodeID(1); err != nil || id != 1 {
		t.Errorf("expected fallback 1, got %d (err: %v)", id, err)
	}

	// Valid NODE_ID test
	t.Setenv("NODE_ID", "42")
	if id, err := resolveNodeID(1); err != nil || id != 42 {
		t.Errorf("expected 42, got %d (err: %v)", id, err)
	}

	// Invalid NODE_ID (non-integer)
	t.Setenv("NODE_ID", "not-a-number")
	if _, err := resolveNodeID(1); err == nil {
		t.Errorf("expected error for non-integer NODE_ID, got nil")
	}

	// Invalid NODE_ID (negative)
	t.Setenv("NODE_ID", "-1")
	if _, err := resolveNodeID(1); err == nil {
		t.Errorf("expected error for negative NODE_ID, got nil")
	}

	// Invalid NODE_ID (out of range > 1023)
	t.Setenv("NODE_ID", "1024")
	if _, err := resolveNodeID(1); err == nil {
		t.Errorf("expected error for NODE_ID > 1023, got nil")
	}

	// POD_NAME test & determinism
	t.Setenv("NODE_ID", "")
	t.Setenv("POD_NAME", "api-78f9d68498-abcde")
	id1, err := resolveNodeID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id1 < 0 || id1 >= 1024 {
		t.Errorf("expected node ID between 0 and 1023, got %d", id1)
	}
	// Determinism check: same POD_NAME should return identical ID
	id1Again, err := resolveNodeID(1)
	if err != nil || id1Again != id1 {
		t.Errorf("expected identical ID %d, got %d (err: %v)", id1, id1Again, err)
	}

	t.Setenv("POD_NAME", "api-78f9d68498-xyz12")
	id2, err := resolveNodeID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id2 < 0 || id2 >= 1024 {
		t.Errorf("expected node ID between 0 and 1023, got %d", id2)
	}
	if id1 == id2 {
		t.Logf("note: both pod names resulted in %d (valid, but ideally distinct)", id1)
	}
}
