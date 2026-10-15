package evaluator

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestLoopClosuresCaptureEachIteration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`let fns = []
repeat i in 3 { fns = fns.insert(fn() { i }) }
fns.map(fn(f) { f() })`, "[1, 2, 3]"},
		{`let fns = []
repeat x in ["a", "b"] { fns = fns.insert(fn() { x }) }
fns.map(fn(f) { f() })`, `["a", "b"]`},
		{`let fns = []
let n = 0
repeat n < 2 {
  n = n + 1
  let current = n
  fns = fns.insert(fn() { current })
}
fns.map(fn(f) { f() })`, "[1, 2]"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}
}

func TestBlockScopesDoNotLeak(t *testing.T) {
	tests := []string{
		"if true { let inner = 1 }\ninner",
		"if false { 1 } else { let inner = 1 }\ninner",
		"repeat i in 2 { let inner = 1 }\ninner",
		"repeat i in 2 { }\ni",
		"repeat x in [1] { }\nx",
		"let n = 0\nrepeat n < 1 { n = n + 1\nlet inner = 1 }\ninner",
		"fn f(a) { let inner = a }\nf(1)\ninner",
		"fn f(a) { a }\nf(1)\na",
	}

	for _, input := range tests {
		errObj, ok := testEval(input).(*object.Error)
		if !ok || !strings.HasPrefix(errObj.Message, "identifier not found") {
			t.Errorf("%q: expected identifier not found, got %s", input, testEval(input).Inspect())
		}
	}
}

func TestBlockScopeAssignments(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"let x = 1\nif true { x = 2 }\nx", 2},
		{"let x = 1\nif true { let x = 5 }\nx", 1},
		{"let total = 0\nrepeat i in 4 { let sq = i * i\ntotal = total + sq }\ntotal", 30},
		{"let n = 0\nrepeat n < 3 { n = n + 1 }\nn", 3},
		{"fn f(a) { let a = a * 10\nreturn a }\nf(2)", 20},
		{"let a = 7\nfn f(a) { a = a + 1\nreturn a }\nf(1) * 10 + a", 27},
	}

	for _, tt := range tests {
		testIntegerObject(t, testEval(tt.input), tt.expected, tt.input)
	}
}
