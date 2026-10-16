package vm

import "testing"

func TestDestructuring(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let [a, b] = [1, 2]\na + b", "3"},
		{"let [a, b, c] = [1, 2]\n[a, b, c]", "[1, 2, null]"},
		{"let [first] = [7, 8, 9]\nfirst", "7"},
		{`let [x, y] = "hi"` + "\nx + y", "hi"},
		{`let {name, age} = {"name": "Ana", "age": 30, "city": "Rio"}` + "\n[name, age]", `["Ana", 30]`},
		{`let {missing} = {"a": 1}` + "\nmissing", "null"},
		{`let pair = [1, 2]` + "\nlet [a, b] = pair", "[1, 2]"},
		{"fn minmax(list) { [list.first(), list.last()] }\nlet [lo, hi] = minmax([3, 5, 9])\nhi - lo", "6"},
		{"fn f() { let [a, b] = [1, 2]\nfn() { a + b } }\nf()()", "3"},
		{"let total = 0\nrepeat pair in [[1, 2], [3, 4]] { let [a, b] = pair\ntotal = total + a * b }\ntotal", "14"},
		{"const [a, b] = [1, 2]\na = 5", "ERROR at line 2, column 3: cannot reassign constant: a"},
		{"const a = 1\nlet [a, b] = [1, 2]", "ERROR at line 2, column 1: cannot reassign constant: a"},
		{"let [a, b] = 5", "ERROR at line 1, column 5: cannot use [] on an integer"},
		{`let {a} = null`, `ERROR at line 1, column 5: cannot read property "a" of null`},
		{`let {a, b} = {"a": 1}` + "\nlet [c, d] = [a, b]\n[c, d]", "[1, null]"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
