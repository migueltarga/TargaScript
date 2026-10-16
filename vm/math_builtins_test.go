package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestFlexibleMathBuiltins(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"max(1, 5)", "5"},
		{"max(1, 5, 3)", "5"},
		{"max([4, 9.5, 2])", "9.5"},
		{"min([4, 2, 2.0])", "2"},
		{"type(min([4, 2]))", "integer"},
		{"min(3, -1, 2)", "-1"},
		{"round(3.14159, 2)", "3.14"},
		{"round(2.5)", "3"},
		{"round(7, 1)", "7"},
		{"type(round(7, 1))", "float"},
		{"range(0, 10, 3)", "[0, 3, 6, 9]"},
		{"range(5, 1, -2)", "[5, 3, 1]"},
		{"range(1, 3)", "[1, 2, 3]"},
		{"range(3, 1, 1)", "[]"},
		{"random([7])", "7"},
		{"random([])", "null"},
		{"max([])", "null"},
		{"random(4, 4)", "4"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestRandomStaysInBounds(t *testing.T) {
	seen := map[int64]bool{}
	for range 200 {
		n, ok := testEval("random(1, 3)").(*object.Integer)
		if !ok || n.Value < 1 || n.Value > 3 {
			t.Fatalf("random(1, 3) = %v", n)
		}
		seen[n.Value] = true
	}
	if len(seen) != 3 {
		t.Errorf("random(1, 3) gave only %v", seen)
	}
	f, ok := testEval("random()").(*object.Float)
	if !ok || f.Value < 0 || f.Value >= 1 {
		t.Errorf("random() = %v", f)
	}
}

func TestFlexibleMathErrors(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"max()", "want an array or two or more numbers"},
		{"max(5)", "max takes an array or two or more numbers, got an integer"},
		{`min(1, "a")`, "arguments to `min` must be numbers, got a string"},
		{"round(1.5, -1)", "must be an integer from 0 to 15"},
		{"random(3, 1)", "needs a <= b"},
		{"random(1.5, 2)", "takes two integers"},
		{"range(1, 5, 0)", "can't be 0"},
		{"range(1, 5.0)", "must be integers"},
	}
	for _, tt := range tests {
		got := testEval(tt.input)
		err, ok := got.(*object.Error)
		if !ok || !strings.Contains(err.Message, tt.expected) {
			t.Errorf("%s: got %v, want an error with %q", tt.input, got, tt.expected)
		}
	}
}
