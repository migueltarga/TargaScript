package evaluator

import (
	"testing"
)

func TestHashLiteralBareKeys(t *testing.T) {
	testIntegerObject(t, testEval(`let name = "x"
let h = {name: 1, "count": 2}
h.name * 10 + h["count"]`), 12, "bare and quoted keys")

	if !testNullObject(t, testEval(`let name = "x"
let h = {name: 1}
h.x`)) {
		t.Errorf("bare key should not be resolved as a variable")
	}

	testIntegerObject(t, testEval(`let h = {1 + 1: 5}
h[2]`), 5, "computed keys are still evaluated")
}

func TestHashInsertionOrder(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`{"e": 5, "a": 1, "d": 4, "b": 2, "c": 3}`, `{"e": 5, "a": 1, "d": 4, "b": 2, "c": 3}`},
		{`let h = {"b": 1, "a": 2}
h.c = 3
h["a"] = 20
h`, `{"b": 1, "a": 20, "c": 3}`},
		{`{"z": 1, "y": 2, "x": 3}.keys()`, `["z", "y", "x"]`},
		{`{"z": 1, "y": 2, "x": 3}.values()`, "[1, 2, 3]"},
		{`{"a": 1, "b": 2, "a": 3}`, `{"a": 3, "b": 2}`},
	}

	for _, tt := range tests {
		// Run several times since map iteration order is randomized
		for i := 0; i < 20; i++ {
			got := testEval(tt.input).Inspect()
			if got != tt.expected {
				t.Fatalf("%q: got=%s, want=%s", tt.input, got, tt.expected)
			}
		}
	}
}
