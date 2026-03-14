package idgen

import (
	"testing"
)

func TestSnowflakeGenerator_Generate(t *testing.T) {
	gen := NewSnowflakeGenerator(1)
	id1 := gen.Generate()
	id2 := gen.Generate()

	if id1 == "" {
		t.Error("Generated ID is empty")
	}
	if id1 == id2 {
		t.Errorf("Generated identical IDs: %s", id1)
	}
	
	t.Logf("Generated IDs: %s, %s", id1, id2)
}
