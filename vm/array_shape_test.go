package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestArrayShapeMethods(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"[1, 2, 3, 4, 5].chunk(2)", "[[1, 2], [3, 4], [5]]"},
		{"[1, 2].chunk(5)", "[[1, 2]]"},
		{"[].chunk(3)", "[]"},
		{"[1, 2, 3].zip([\"a\", \"b\"])", `[[1, "a"], [2, "b"]]`},
		{"[].zip([1])", "[]"},
		{`[["a", 1], ["b", 2]].toObject()`, `{"a": 1, "b": 2}`},
		{`{"x": 1, "y": 2}.entries().map(([k, v]) => [k, v * 10]).toObject()`, `{"x": 10, "y": 20}`},
		{`["a", "b"].zip([1, 2]).toObject()`, `{"a": 1, "b": 2}`},
		{"let a = [1, 2, 3]; let c = a.chunk(2); c[0][0] = 9; a", "[1, 2, 3]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestArrayShapeErrors(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"[1].chunk(0)", "the size for chunk must be a positive integer, got 0"},
		{"[1].chunk()", "want 1"},
		{"[1].zip(2)", "zip needs an array, got an integer"},
		{"[1].toObject()", "toObject needs [key, value] pairs, got 1"},
		{"[[[1], 2]].toObject()", "an array can't be an object key"},
	}
	for _, tt := range tests {
		got := testEval(tt.input)
		err, ok := got.(*object.Error)
		if !ok || !strings.Contains(err.Message, tt.expected) {
			t.Errorf("%s: got %v, want an error with %q", tt.input, got, tt.expected)
		}
	}
}
