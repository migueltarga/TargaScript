package evaluator

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestMaxNesting(t *testing.T) {
	SetMaxNesting(200)
	defer SetMaxNesting(0)

	recursive := "fn d(n) { if n == 0 { return 0 }\nreturn d(n - 1) + 1 }\n"

	errObj, ok := testEval(recursive + "d(1000)").(*object.Error)
	if !ok {
		t.Fatalf("expected an error for deep recursion")
	}
	if errObj.Message != "maximum call depth exceeded" {
		t.Errorf("wrong error: %q", errObj.Message)
	}

	// The count unwinds after the error, so shallower calls still work
	testIntegerObject(t, testEval(recursive+"d(20)"), 20)
	if nesting != 0 {
		t.Errorf("nesting not reset: %d", nesting)
	}
}
