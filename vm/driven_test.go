package vm

import "testing"

// map, filter and reduce give the same results when the VM runs their
// functions itself, as for functions that call nothing
func TestDrivenCallbacks(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"[1, 2, 3].map(x => x * 2)", "[2, 4, 6]"},
		{"[1, 2, 3].map((x, i) => x * i)", "[0, 2, 6]"},
		{"[1, 2, 3].map((x, i, z) => z)", "[null, null, null]"},
		{"[5, 6, 7].filter((x, i) => i != 1)", "[5, 7]"},
		{"[1, 2, 3].reduce((a, b) => a + b, 0)", "6"},
		{"[1, 2, 3].reduce((a, b, i) => a + b * i, 0)", "8"},
		{"[1, 2].reduce(a => a, 9)", "9"},
		{"[].map(x => 1)", "[]"},
		{"[].reduce((a, b) => a, 7)", "7"},
		{"[[1, 2], [3]].map(it.map(it * 2))", "[[2, 4], [6]]"},
		{"let xs = [1, 2]\n[xs.map(x => { xs[0] = 9\nx }), xs]", "[[1, 2], [9, 2]]"},
		{"let t = 0\ntry { [1, 2].map(x => { throw \"boom\" }) } catch e { t = e.message }\nt", "boom"},
		{"fn f(xs) { xs.map(x => x + 1).filter(x => x > 2).reduce((a, b) => a + b, 0) }\nf([1, 2, 3])", "7"},
		// Functions that call others go the evaluator's way
		{"fn twice(x) { x * 2 }\n[1, 2].map(x => twice(x))", "[2, 4]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}
