package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestArraySearchMethods(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[1, 2, 3].contains(2)", "true"},
		{"[1, 2, 3].contains(5)", "false"},
		{`[1, "2"].contains("1")`, "false"},
		{"[1, 2.0].contains(2)", "true"},
		{"[1, 2, 3, 2].indexOf(2)", "1"},
		{`["a", "b"].indexOf("z")`, "-1"},
		{"[1, 5, 8].find(fn(x) { x > 3 })", "5"},
		{"[1, 2].find(fn(x) { x > 3 })", "null"},
		{"[1, 2, 3].find(toString)", "1"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}
}

func TestArraySlice(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[1, 2, 3, 4].slice(1)", "[2, 3, 4]"},
		{"[1, 2, 3, 4].slice(1, 3)", "[2, 3]"},
		{"[1, 2, 3, 4].slice(-2)", "[3, 4]"},
		{"[1, 2, 3, 4].slice(0, -1)", "[1, 2, 3]"},
		{"[1, 2, 3].slice(5)", "[]"},
		{"[1, 2, 3].slice(2, 1)", "[]"},
		{"[1, 2, 3].slice(-10, 10)", "[1, 2, 3]"},
		{"let a = [1, 2, 3]\nlet b = a.slice(0)\nb[0] = 9\na", "[1, 2, 3]"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}
}

func TestArraySortAndReverse(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[3, 1, 2].sort()", "[1, 2, 3]"},
		{"[2.5, 1, 3].sort()", "[1, 2.5, 3]"},
		{`["pear", "apple", "fig"].sort()`, `["apple", "fig", "pear"]`},
		{"[1, 3, 2].sort(fn(a, b) { b - a })", "[3, 2, 1]"},
		{`let people = [{"n": "b", "age": 30}, {"n": "a", "age": 25}, {"n": "c", "age": 30}]
people.sort(fn(a, b) { a.age - b.age }).map(fn(p) { p.n })`, `["a", "b", "c"]`},
		{"let a = [3, 1, 2]\na.sort()\na", "[1, 2, 3]"},
		{"[].sort()", "[]"},
		{"[1, 2, 3].reverse()", "[3, 2, 1]"},
		{"let a = [1, 2]\na.reverse()\na", "[2, 1]"},
		{"[].reverse()", "[]"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}
}

func TestArrayMethodErrors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`[1, "a"].sort()`, `cannot sort: cannot use "<" with a string and an integer`},
		{"[true, false].sort()", `cannot sort: cannot use "<" with a boolean and a boolean`},
		{`[1, 2].sort(fn(a, b) { "x" })`, "compare function must return a number, got STRING"},
		{"[1, 2].sort(fn(a, b) { missing })", `"missing" is not defined`},
		{"[1].slice(\"a\")", "arguments to array.slice() must be INTEGER, got STRING"},
		{"[1].find(1)", "array.find() takes one FUNCTION argument"},
		{"[1].contains()", "wrong number of arguments for array.contains(): got 0, want 1"},
	}

	for _, tt := range tests {
		errObj, ok := testEval(tt.input).(*object.Error)
		if !ok {
			t.Errorf("%q: expected an error, got %s", tt.input, testEval(tt.input).Inspect())
			continue
		}
		if !strings.Contains(errObj.Message, tt.expected) {
			t.Errorf("%q: error %q does not contain %q", tt.input, errObj.Message, tt.expected)
		}
	}
}
