package vm

import "testing"

func TestRestParameters(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"fn sum(...numbers) { numbers.reduce(fn(a, b) { a + b }, 0) }\nsum(1, 2, 3)", "6"},
		{"fn sum(...numbers) { numbers }\nsum()", "[]"},
		{"fn f(first, ...rest) { [first, rest] }\nf(1, 2, 3)", "[1, [2, 3]]"},
		{"fn f(first, ...rest) { [first, rest] }\nf(1)", "[1, []]"},
		{"fn f(first, ...rest) { [first, rest] }\nf()", "[null, []]"},
		{"fn f(a, b = 5, ...rest) { [a, b, rest] }\nf(1)", "[1, 5, []]"},
		{"fn f(...all) { all.length }\nlet args = [1, 2, 3, 4]\nf(...args)", "4"},
		{"let log = fn(level, ...parts) { \"[${level}] \" + parts.join(\" \") }\nlog(\"info\", \"a\", \"b\")", "[info] a b"},
		{"fn f(...xs) { fn() { xs.length } }\nf(1, 2)()", "2"},
		{"[1, 2].map(fn(...all) { all.length })", "[1, 1]"},
		{"fn f(a, ...rest) { 1 }\nf", "fn f(a, ...rest)"},
		{"fn f(...rest, a) { 1 }", `ERROR: parser errors: line 1:13: the ...rest parameter must be the last one`},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
