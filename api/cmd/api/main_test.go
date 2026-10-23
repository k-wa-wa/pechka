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
