package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestRangeOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1..5", "[1, 2, 3, 4, 5]"},
		{"3..3", "[3]"},
		{"5..1", "[]"},
		{"-2..1", "[-2, -1, 0, 1]"},
		{"let n = 3\n1..n + 1", "[1, 2, 3, 4]"},
		{"(1..3).map(fn(x) { x * 10 })", "[10, 20, 30]"},
		{"(1..4).length", "4"},
		{"range(1, 3)", "[1, 2, 3]"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}
}

func TestRepeatOverRange(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"let s = 0\nrepeat i in 1..4 { s = s + i }\ns", 10},
		{"let s = 0\nrepeat i in 3..1 { s = s + 1 }\ns", 0},
		{"let s = 0\nrepeat i in 0..2 { s = s * 10 + i }\ns", 12},
		// Not materialized, so a huge range is fine as long as the loop stops
		{"let last = 0\nrepeat i in 1..1000000000000 {\n  last = i\n  if i == 5 { break }\n}\nlast", 5},
	}

	for _, tt := range tests {
		testIntegerObject(t, testEval(tt.input), tt.expected, tt.input)
	}
}

func TestRangeErrors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1..2.5", "range bounds must be integers, got an integer and a float"},
		{`"a".."c"`, "range bounds must be integers, got a string and a string"},
		{"repeat i in 1..\"x\" { }", "range bounds must be integers"},
		{"1..100000000", "range too large"},
		{"range(1, 100000000)", "range too large"},
	}

	for _, tt := range tests {
		errObj, ok := testEval(tt.input).(*object.Error)
		if !ok {
			t.Errorf("%q: expected an error", tt.input)
			continue
		}
		if !strings.Contains(errObj.Message, tt.expected) {
			t.Errorf("%q: error %q does not contain %q", tt.input, errObj.Message, tt.expected)
		}
	}
}
