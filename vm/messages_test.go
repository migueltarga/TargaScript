package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestErrorSuggestions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let total = 1\ntotl", `"totl" is not defined. Did you mean "total"?`},
		{"let count = 1\nConut", `"Conut" is not defined. Did you mean "count"?`},
		{"prnt(1)", `"prnt" is not defined. Did you mean "print"?`},
		{"lenght([1])", `"lenght" is not defined`},
		{"toStrng(1)", `"toStrng" is not defined. Did you mean "toString"?`},
		{"foobar", `"foobar" is not defined`},
		{"let a = 1\nb", `"b" is not defined`},
		{`"abc".lenght`, `strings have no property "lenght". Did you mean "length"?`},
		{`[1].lenght`, `arrays have no property "lenght". Did you mean "length"?`},
		{`"abc".uper()`, `strings have no method "uper". Did you mean "upper"?`},
		{`[1, 2].fliter(fn(x) { x })`, `arrays have no method "fliter". Did you mean "filter"?`},
		{`[1, 2].total()`, `arrays have no method "total"`},
		{`let h = {"greet": fn() { 1 }}` + "\nh.gret()", `object has no method "gret". Did you mean "greet"?`},
		{`let h = {"name": "x"}` + "\nh.name()", `property "name" is a string, not a function`},
		{"let n = 5\nn.upper()", `cannot call method "upper" on an integer`},
		{`1 + true`, `cannot use "+" with an integer and a boolean`},
		{`-"a"`, `cannot use "-" on a string`},
		{`5[0]`, `cannot use [] on an integer`},
		{`let x = 5` + "\nx()", `cannot call an integer, it is not a function`},
	}

	for _, tt := range tests {
		errObj, ok := testEval(tt.input).(*object.Error)
		if !ok {
			t.Errorf("%q: expected an error, got %s", tt.input, testEval(tt.input).Inspect())
			continue
		}
		if errObj.Message != tt.expected {
			t.Errorf("%q:\n got: %s\nwant: %s", tt.input, errObj.Message, tt.expected)
		}
	}
}
