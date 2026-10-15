package evaluator

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestRandom(t *testing.T) {
	seen := map[float64]bool{}

	for i := 0; i < 100; i++ {
		result, ok := testEval("random()").(*object.Float)
		if !ok {
			t.Fatalf("random() did not return a Float")
		}
		if result.Value < 0 || result.Value >= 1 {
			t.Fatalf("random() out of range: %g", result.Value)
		}
		seen[result.Value] = true
	}

	if len(seen) < 90 {
		t.Errorf("random() returned only %d distinct values out of 100", len(seen))
	}
}
