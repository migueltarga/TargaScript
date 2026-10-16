package vm

import "testing"

func TestPatternRenames(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`let {name: who, age} = {name: "Ana", age: 3}; [who, age]`, `["Ana", 3]`},
		{`let {"first name": first} = {"first name": "Bia"}; first`, `"Bia"`},
		{`let {a: x = 5} = {}; x`, "5"},
		{`let {a: x, ...rest} = {a: 1, b: 2}; [x, rest]`, `[1, {"b": 2}]`},
		{`const {id: key} = {id: 7}; key`, "7"},
		{`fn f({name: who}) { who }; f({name: "Caio"})`, `"Caio"`},
		{`let f = ({name: who}) => who; f({name: "Dani"})`, `"Dani"`},
		{`let f = ({"x y": v}) => v; f({"x y": 1})`, "1"},
		{`let total = 0; repeat {price: p} in [{price: 2}, {price: 3}] { total += p }; total`, "5"},
	}
	for _, tt := range tests {
		got := testEval(tt.input)
		inspected := got.Inspect()
		if got.Type() == "STRING" {
			inspected = `"` + inspected + `"`
		}
		if inspected != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, inspected, tt.expected)
		}
	}
}
