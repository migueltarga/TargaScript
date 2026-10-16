package vm

import "testing"

func TestPatternDefaults(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`let {port = 8080, host} = {host: "x"}; [port, host]`, `[8080, "x"]`},
		{`let {port = 8080} = {port: 1}; port`, "1"},
		{`let {port = 8080} = {port: null}; port`, "8080"},
		{`let {flag = true} = {flag: false}; flag`, "false"},
		{"let [a, b = a * 10] = [1]; [a, b]", "[1, 10]"},
		{"let [a = 1, ...rest] = []; [a, rest]", "[1, []]"},
		{`const {mode = "dev"} = {}; mode`, `"dev"`},
		{"fn f({x, y = 2}) { x + y }; f({x: 1})", "3"},
		{"let f = ([x, y = 2]) => x + y; f([1])", "3"},
		{"let total = 0; repeat [n, times = 1] in [[2], [3, 2]] { total += n * times }; total", "8"},
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
