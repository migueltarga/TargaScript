package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestInOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`"b" in ["a", "b"]`, true},
		{`"c" in ["a", "b"]`, false},
		{`[1, 2] in [[1, 2], [3]]`, true},
		{`1.0 in [1]`, true},
		{`"name" in {name: "Ana"}`, true},
		{`"age" in {name: "Ana"}`, false},
		{`[1] in {name: "Ana"}`, false},
		{`"ell" in "hello"`, true},
		{`"xyz" in "hello"`, false},
		{`5 in 1..10`, true},
		{`11 in 1..10`, false},
		{`2.5 in 1..3`, true},
		{`let x = 4; x in 1..x`, true},
		{`!("a" in ["b"])`, true},
		{`1 + 1 in [2]`, true},
		{`"a" in ["a"] == true`, true},
		{`"a" in ["a"] && "b" in ["b"]`, true},
	}
	for _, tt := range tests {
		result, ok := testEval(tt.input).(*object.Boolean)
		if !ok || result.Value != tt.expected {
			t.Errorf("%s: got %v, want %t", tt.input, testEval(tt.input), tt.expected)
		}
	}
}

func TestInOperatorErrors(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`1 in 5`, `cannot use "in" with an integer`},
		{`1 in "abc"`, "cannot look for an integer in a string"},
	}
	for _, tt := range tests {
		err, ok := testEval(tt.input).(*object.Error)
		if !ok || !strings.Contains(err.Message, tt.expected) {
			t.Errorf("%s: got %v, want an error with %q", tt.input, testEval(tt.input), tt.expected)
		}
	}
}

func TestRepeatStillUsesIn(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let s = 0; repeat x in [1, 2, 3] { s += x }; s", "6"},
		{"let s = 0; repeat [a, b] in [[1, 2], [3, 4]] { s += a * b }; s", "14"},
		{"let s = 0; repeat {a} in [{a: 1}, {a: 2}] { s += a }; s", "3"},
		{"let s = 0; repeat k, v in {a: 1, b: 2} { s += v }; s", "3"},
		{"let s = []; repeat x in [1, 2] { if x in [2] { s.insert(x) } }; s", "[2]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}
