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
	if id := resolveNodeID(1); id != 1 {
		t.Errorf("expected fallback 1, got %d", id)
	}

	// NODE_ID test
	t.Setenv("NODE_ID", "42")
	if id := resolveNodeID(1); id != 42 {
		t.Errorf("expected 42, got %d", id)
	}

	// POD_NAME test
	t.Setenv("NODE_ID", "")
	t.Setenv("POD_NAME", "api-78f9d68498-abcde")
	id1 := resolveNodeID(1)
	if id1 < 0 || id1 >= 1024 {
		t.Errorf("expected node ID between 0 and 1023, got %d", id1)
	}

	t.Setenv("POD_NAME", "api-78f9d68498-xyz12")
	id2 := resolveNodeID(1)
	if id2 < 0 || id2 >= 1024 {
		t.Errorf("expected node ID between 0 and 1023, got %d", id2)
	}
	if id1 == id2 {
		t.Logf("note: both pod names resulted in %d (valid, but ideally distinct)", id1)
	}
}
