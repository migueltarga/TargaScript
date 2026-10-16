package vm

import "testing"

func TestArrowFunctions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[1, 2, 3].map(x => x * 2)", "[2, 4, 6]"},
		{"[1, 2, 3, 4].filter(n => n % 2 == 0)", "[2, 4]"},
		{"let add = (a, b) => a + b\nadd(2, 3)", "5"},
		{"let f = () => 42\nf()", "42"},
		{"let f = (a, b = 10) => a + b\n[f(1), f(1, 2)]", "[11, 3]"},
		{"let f = x => {\n  let y = x * 2\n  y + 1\n}\nf(5)", "11"},
		{"let adder = x => y => x + y\nadder(2)(3)", "5"},
		{"[[1, 2], [3]].map(list => list.map(n => n * 10))", "[[10, 20], [30]]"},
		{"[3, 1, 2].sortBy(n => -n)", "[3, 2, 1]"},
		{"let total = 0\n[1, 2].map(n => total += n)\ntotal", "3"},
		{"let x = 3\nmatch x { 1 => \"one\"\n x => \"same\" }", "same"},
		{"match 2 { 2 => (n => n + 1)(5), _ => 0 }", "6"},
		{"let f = (a) => a\nf(7)", "7"},
		{"(1 + 2) * 3", "9"},
		{"(1, 2)", "ERROR: parser errors: line 1:1: expected one expression in (), found 2 separated by commas"},
		{"(1, 2) => 3", "ERROR: parser errors: line 1:1: expected parameter names before =>"},
		{"()", "ERROR: parser errors: line 1:1: expected an expression in ()"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}

func TestArrowFunctionPatterns(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[[1, 2], [3, 4]].map(([a, b]) => a * b)", "[2, 12]"},
		{`[{"n": "x", "v": 1}].map(({n, v}) => n + v)`, `["x1"]`},
		{`[{"a": {"b": 2}}].map(({a}) => a.b)`, "[2]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
