package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestStringInterpolation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`let name = "Ana"
"Olá ${name}!"`, "Olá Ana!"},
		{`let age = 30
"Next year: ${age + 1}"`, "Next year: 31"},
		{`"${1}${2} and ${3}"`, "12 and 3"},
		{`let h = {"k": "v"}
"value: ${h["k"]}"`, "value: v"},
		{`let p = {"name": "Ana", "tags": ["a", "b"]}
"${p.name} has ${p.tags.length} tags: ${p.tags.join(", ")}"`, "Ana has 2 tags: a, b"},
		{`"inner: ${"x" + "y"}"`, "inner: xy"},
		{`"braces: ${ {"a": 1}.a }"`, "braces: 1"},
		{`"value: ${true}, ${1.5}, ${[1, 2]}"`, "value: true, 1.5, [1, 2]"},
		{`fn greet(n) { return "Hi ${n}" }
greet("Bo")`, "Hi Bo"},
		{`"literal: \${name}"`, "literal: ${name}"},
		{`let x = 1
"tab:\t${x}\n"`, "tab:\t1\n"},
		{`"no interpolation: $ {x} $x"`, "no interpolation: $ {x} $x"},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		str, ok := evaluated.(*object.String)
		if !ok {
			t.Errorf("%q: expected String, got %T (%s)", tt.input, evaluated, evaluated.Inspect())
			continue
		}
		if str.Value != tt.expected {
			t.Errorf("%q: got=%q, want=%q", tt.input, str.Value, tt.expected)
		}
	}
}

func TestStringInterpolationRuntimeError(t *testing.T) {
	errObj, ok := testEval(`"${missing}"`).(*object.Error)
	if !ok {
		t.Fatalf("expected an error")
	}
	if errObj.Message != `"missing" is not defined` {
		t.Errorf("wrong error message: %q", errObj.Message)
	}
}
