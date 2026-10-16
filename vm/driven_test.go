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

func TestDrivenSearches(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"[5, 6, 7].find(x => x > 5)", "6"},
		{"[5, 6, 7].find((x, i) => i == 2)", "7"},
		{"[5, 6].find(x => x > 9)", "null"},
		{"[].find(x => true)", "null"},
		{"[1, 2].some(x => x > 1)", "true"},
		{"[1, 2].some(x => x > 5)", "false"},
		{"[].some(x => true)", "false"},
		{"[1, 2].every(x => x > 0)", "true"},
		{"[1, 2].every((x, i) => i == 0)", "false"},
		{"[].every(x => false)", "true"},
		{"[1, 2, 3].count(x => x > 1)", "2"},
		{"[1, 2, 3].count((x, i) => i > 0)", "2"},
		{"[1, 2, 1].count(1)", "2"},
		{"[].count(x => true)", "0"},
		// They stop at the element that decides
		{"let n = 0\n[1, 2, 3].find(x => { n += 1\nx == 2 })\nn", "2"},
		{"let n = 0\n[1, 2, 3].every(x => { n += 1\nx < 2 })\nn", "2"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestDrivenSortBy(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"[3, 1, 2].sortBy(x => x)", "[1, 2, 3]"},
		{"[3, 1, 2].sortBy(x => -x)", "[3, 2, 1]"},
		{"[\"bb\", \"a\", \"ccc\"].sortBy(s => s.length)", `["a", "bb", "ccc"]`},
		{"[3, 1, 2].sortBy((x, i) => -i)", "[2, 1, 3]"},
		{"[{k: 1, i: 0}, {k: 0, i: 1}, {k: 1, i: 2}].sortBy(it.k).map(it.i)", "[1, 0, 2]"},
		{"let xs = [2, 1]\nxs.sortBy(x => x)\nxs", "[1, 2]"},
		{"[].sortBy(x => x)", "[]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
	if result := testEval("[1, \"a\"].sortBy(x => x)"); !isError(result) {
		t.Errorf("keys that can't be compared should fail, got %s", result.Inspect())
	}
}
