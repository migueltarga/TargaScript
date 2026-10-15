package evaluator

import (
	"bytes"
	"os"
	"testing"
)

func TestPrintQuotesOnlyNestedStrings(t *testing.T) {
	var out bytes.Buffer
	SetOutput(&out)
	defer SetOutput(os.Stdout)

	testEval(`print("top", ["a", 1], {"k": "v"}, ["x", "y"].join(","))`)

	expected := "top [\"a\", 1] {\"k\": \"v\"} x,y\n"
	if out.String() != expected {
		t.Errorf("got=%q, want=%q", out.String(), expected)
	}
}
