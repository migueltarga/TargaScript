package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestThrowAsAnExpression(t *testing.T) {
	tests := []struct{ input, message string }{
		{`let port = null ?? throw "no port"`, "no port"},
		{`match 3 { 1 => "one", _ => throw "unknown ${3}" }`, "unknown 3"},
		{`let f = x => throw "bad ${x}"; f(1)`, "bad 1"},
		{`let v = true && throw "both"`, "both"},
	}
	for _, tt := range tests {
		err, ok := testEval(tt.input).(*object.Error)
		if !ok || err.Message != tt.message {
			t.Errorf("%s: got %v, want an error %q", tt.input, testEval(tt.input), tt.message)
		}
	}

	caught := testEval(`try { null ?? throw "x" } catch e { "caught " + e.message }`)
	if str, ok := caught.(*object.String); !ok || str.Value != "caught x" {
		t.Errorf("got %s", caught.Inspect())
	}
	if got := testEval(`1 ?? throw "never"`).Inspect(); got != "1" {
		t.Errorf("throw ran when it shouldn't: %s", got)
	}
}
