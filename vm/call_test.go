package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestMaxCallDepth(t *testing.T) {
	evaluated := testEval("fn f(n) { return f(n + 1) }\nf(0)")

	errObj, ok := evaluated.(*object.Error)
	if !ok {
		t.Fatalf("expected error, got=%T (%+v)", evaluated, evaluated)
	}
	if errObj.Message != "maximum call depth exceeded (10000)" {
		t.Errorf("wrong error message: %q", errObj.Message)
	}
	if errObj.Line == 0 {
		t.Errorf("error has no line number")
	}

	// Deep but bounded recursion still works
	testIntegerObject(t, testEval(`
fn count(n) {
  if n == 0 { return 0 }
  return 1 + count(n - 1)
}
count(5000)`), 5000, "deep recursion")
}

func TestCallErrorsHavePosition(t *testing.T) {
	evaluated := testEval("let x = 5\n\nx(1)")

	errObj, ok := evaluated.(*object.Error)
	if !ok {
		t.Fatalf("expected error, got=%T (%+v)", evaluated, evaluated)
	}
	if errObj.Message != "cannot call an integer, it is not a function" {
		t.Errorf("wrong error message: %q", errObj.Message)
	}
	if errObj.Line != 3 {
		t.Errorf("wrong line. got=%d, want=3", errObj.Line)
	}
}

// Functions at the top of a program can be used above their declaration
func TestTopLevelFunctionsFirst(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let x = f()\nfn f() { 3 }\nx", "3"},
		{"let t = {g: f}\nfn f() { 2 }\nt.g()", "2"},
		{"fn f() { 1 }\nlet a = f()\nfn f() { 2 }\n[a, f()]", "[1, 2]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}
