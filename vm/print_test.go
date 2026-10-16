package vm

import (
	"bytes"
	"github.com/migueltarga/TargaScript/evaluator"
	"os"
	"testing"
)

func TestPrintQuotesOnlyNestedStrings(t *testing.T) {
	var out bytes.Buffer
	evaluator.SetOutput(&out)
	defer evaluator.SetOutput(os.Stdout)

	testEval(`print("top", ["a", 1], {"k": "v"}, ["x", "y"].join(","))`)

	expected := "top [\"a\", 1] {\"k\": \"v\"} x,y\n"
	if out.String() != expected {
		t.Errorf("got=%q, want=%q", out.String(), expected)
	}
}
