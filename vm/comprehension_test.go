package vm

import "testing"

func TestListComprehension(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[n * 2 repeat n in [1, 2, 3]]", "[2, 4, 6]"},
		{"[n repeat n in 1..10 if n % 2 == 0]", "[2, 4, 6, 8, 10]"},
		{"[n repeat n in []]", "[]"},
		{`[c.upper() repeat c in "ab"]`, `["A", "B"]`},
		{`[k + "=" + v repeat k, v in {"a": 1, "b": 2}]`, `["a=1", "b=2"]`},
		{"[i * v repeat i, v in [5, 6]]", "[0, 6]"},
		{"[x + y repeat [x, y] in [[1, 2], [3, 4]]]", "[3, 7]"},
		{"[name repeat {name} in [{name: \"Ana\"}, {name: \"Bia\"}]]", `["Ana", "Bia"]`},
		{"[[x, y] repeat x in 1..2 repeat y in 1..2 if x != y]", "[[1, 2], [2, 1]]"},
		{"[x * y repeat x in 1..3 if x != 2 repeat y in [10, 100]]", "[10, 100, 30, 300]"},
		{"let k = 10\n[x + k repeat x in [1, 2]]", "[11, 12]"},
		{"let double = xs => [x * 2 repeat x in xs]\ndouble([1, 2])", "[2, 4]"},
		{"fn f(xs) { [x * 2 repeat x in xs] }\nf([1, 2]) + f([3])", "[2, 4, 6]"},
		{"fn f(xs) { [() => x repeat x in xs] }\nf([1, 2]).map(g => g())", "[1, 2]"},
		{"[\n    x * 2\n    repeat x in [1, 2]\n    if x > 1\n]", "[4]"},
		{"let squares = [x * x repeat x in 1..3]\nsquares.sum()", "14"},
		{"[x repeat x in [1]]\nx", `ERROR at line 2, column 1: "x" is not defined`},
		{"[x repeat 3]", `ERROR: parser errors: line 1:4: expected a name before "in" in the list comprehension, like [x * 2 repeat x in list]`},
		{"[x repeat x < 3]", `ERROR: parser errors: line 1:4: expected a name before "in" in the list comprehension, like [x * 2 repeat x in list]`},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
