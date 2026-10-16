package vm

import "testing"

func TestOwnPropertiesComeFirst(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`parseJSON("{\"title\": \"x\", \"length\": 300}").length`, "300"},
		{"{a: 1, b: 2}.length", "2"},
		{"{}.length", "0"},
		{"let o = {keys: fn() { \"mine\" }}; o.keys()", `"mine"`},
		{"let o = {length: fn() { 9 }}; o.length()", "9"},
		{"let o = {has: fn(k) { k == \"x\" }}; [o.has(\"x\"), o.has(\"y\")]", "[true, false]"},
		// Data that isn't a function doesn't hide the methods
		{"let o = {keys: [1]}; o.keys()", `["keys"]`},
		{"let o = {values: 2}; o.values()", "[2]"},
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
