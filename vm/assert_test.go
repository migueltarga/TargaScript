package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestAssert(t *testing.T) {
	passing := []string{
		`assert(true)`,
		`assert(1 + 1 == 2, "math")`,
		`assertEqual(1 + 1, 2)`,
		`assertEqual([1, {a: 2}], [1, {a: 2}])`,
		`assertEqual(1, 1.0, "numbers")`,
	}
	for _, input := range passing {
		if got := testEval(input); got.Type() == object.ERROR_OBJ {
			t.Errorf("%s: got %s", input, got.Inspect())
		}
	}

	failing := []struct{ input, expected string }{
		{`assert(false)`, "assertion failed"},
		{`assert(0, "no items")`, "assertion failed: no items"},
		{`assertEqual(1 + 1, 3)`, "assertion failed: expected 3, got 2"},
		{`assertEqual("a", "b", "names")`, `assertion failed: names: expected "b", got "a"`},
		{`assert()`, "wrong number of arguments for assert: got 0, want 1 or 2"},
	}
	for _, tt := range failing {
		got := testEval(tt.input)
		err, ok := got.(*object.Error)
		if !ok || err.Message != tt.expected {
			t.Errorf("%s: got %v, want %q", tt.input, got, tt.expected)
		}
	}

	caught := testEval(`try { assert(false, "x") } catch e { e.message }`)
	if str, ok := caught.(*object.String); !ok || str.Value != "assertion failed: x" {
		t.Errorf("catching an assertion: got %s", caught.Inspect())
	}
}
