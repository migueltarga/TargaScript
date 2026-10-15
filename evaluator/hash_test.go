package evaluator

import (
	"testing"
)

func TestHashLiteralBareKeys(t *testing.T) {
	testIntegerObject(t, testEval(`let name = "x"
let h = {name: 1, "count": 2}
h.name * 10 + h["count"]`), 12, "bare and quoted keys")

	if !testNullObject(t, testEval(`let name = "x"
let h = {name: 1}
h.x`)) {
		t.Errorf("bare key should not be resolved as a variable")
	}

	testIntegerObject(t, testEval(`let h = {1 + 1: 5}
h[2]`), 5, "computed keys are still evaluated")
}
