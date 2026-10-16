package vm

import "testing"

func TestKeywordsAsKeys(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"{if: 1, else: 2, in: 3}", `{"if": 1, "else": 2, "in": 3}`},
		{"let o = {load: fn() { 7 }}; o.load()", "7"},
		{"let o = {match: 1, catch: 2}; o.match + o.catch", "3"},
		{"let {load: f, in: i} = {load: 1, in: 2}; [f, i]", "[1, 2]"},
		{"let f = ({if: c}) => c; f({if: true})", "true"},
		// true, false and null stay values
		{`{true: "yes", false: "no"}[true]`, `"yes"`},
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
