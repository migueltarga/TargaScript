package evaluator

import "testing"

func TestNullLiteral(t *testing.T) {
	testNullObject(t, testEval("null"))
	testNullObject(t, testEval("let x = null\nx"))
	testBooleanObject(t, testEval("null == null"), true)
	testBooleanObject(t, testEval("let h = {}\nh.missing == null"), true)
	testBooleanObject(t, testEval("!null"), true)

	if got := testEval(`"value: " + null`).Inspect(); got != "value: null" {
		t.Errorf("got=%q", got)
	}
}
