package vm

import "testing"

func TestMoreArrayMethods(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[1, 2, 3].some(fn(x) { x > 2 })", "true"},
		{"[1, 2, 3].some(fn(x) { x > 5 })", "false"},
		{"[].some(fn(x) { true })", "false"},
		{"[1, 2, 3].every(fn(x) { x > 0 })", "true"},
		{"[1, 2, 3].every(fn(x) { x > 1 })", "false"},
		{"[].every(fn(x) { false })", "true"},
		{"[1, 2, 1, 1].count(1)", "3"},
		{"[1, 2, 3, 4].count(fn(x) { x % 2 == 0 })", "2"},
		{"[1, 2, 3].sum()", "6"},
		{"[1, 2.5].sum()", "3.5"},
		{"[].sum()", "0"},
		{`[1, "a"].sum()`, `ERROR at line 1, column 9: array.sum() can only add numbers, got a string`},
		{"[9223372036854775807, 1].sum()", "ERROR at line 1, column 25: integer overflow"},
		{"[3, 1, 2].min()", "1"},
		{"[3, 1, 2].max()", "3"},
		{`["b", "a", "c"].max()`, "c"},
		{"[].min()", "null"},
		{`[1, "a"].min()`, `ERROR at line 1, column 9: cannot find the min: cannot use "<" with a string and an integer`},
		{"[1, 2, 1, 3, 2].unique()", "[1, 2, 3]"},
		{"[[1], [1], [2]].unique()", "[[1], [2]]"},
		{"[1, [2, [3, [4]]]].flatten()", "[1, 2, [3, [4]]]"},
		{"[1, [2, [3, [4]]]].flatten(10)", "[1, 2, 3, 4]"},
		{`["apple", "avocado", "banana"].groupBy(fn(w) { w[0] })`, `{"a": ["apple", "avocado"], "b": ["banana"]}`},
		{`[1, 2, 3, 4].groupBy(fn(n) { n % 2 == 0 })`, "{false: [1, 3], true: [2, 4]}"},
		{`[{"n": "b", "a": 2}, {"n": "a", "a": 1}].sortBy(fn(p) { p.a }).map(fn(p) { p.n })`, `["a", "b"]`},
		{`["bb", "a", "ccc"].sortBy(fn(s) { s.length })`, `["a", "bb", "ccc"]`},
		{"[1].some(5)", "ERROR at line 1, column 4: array.some() takes a function"},
		{"[1].summ()", `ERROR at line 1, column 4: arrays have no method "summ". Did you mean "sum"?`},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
