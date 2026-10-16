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

func TestPrintIsAFunction(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`"hi" |> print`, "hi\n"},
		{`[1, 2].map(print)`, "1\n2\n"},
		{`let say = print; say("a", 1)`, "a 1\n"},
		{`let p = print("x"); print(p)`, "x\nnull\n"},
		{`let print = fn(x) { 1 }; print("hidden")`, ""},
	}
	for _, tt := range tests {
		var out bytes.Buffer
		evaluator.SetOutput(&out)
		testEval(tt.input)
		evaluator.SetOutput(os.Stdout)
		if out.String() != tt.expected {
			t.Errorf("%s: printed %q, want %q", tt.input, out.String(), tt.expected)
		}
	}
}
