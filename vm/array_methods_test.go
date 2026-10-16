package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestArrayMethodsAcceptBuiltins(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`[1, 2, 3].map(toString).join("-")`, "1-2-3"},
		{`[-1, 2, -3].map(abs)`, "[1, 2, 3]"},
		{`[1, 0, 2].filter(toString)`, "[1, 0, 2]"},
		{`[3, 7, 5].reduce(max, 0)`, "7"},
		{`[1, 2].map(fn(x) { x * 2 })`, "[2, 4]"},
	}

	for _, tt := range tests {
		got := testEval(tt.input)
		if got.Inspect() != tt.expected {
			t.Errorf("%s: got=%s, want=%s", tt.input, got.Inspect(), tt.expected)
		}
	}

	if _, ok := testEval(`[1].map(5)`).(*object.Error); !ok {
		t.Errorf("expected an error for a non-callable argument")
	}
}
