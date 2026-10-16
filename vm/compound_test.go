package vm

import "testing"

func TestCompoundAssignment(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let x = 1\nx += 2\nx", "3"},
		{"let x = 10\nx -= 3\nx *= 2\nx /= 7\nx", "2"},
		{"let x = 7\nx %= 4\nx", "3"},
		{"let x = 3\nx **= 2\nx", "9"},
		{`let s = "a"` + "\ns += \"b\"\ns", "ab"},
		{"let x = 1\nlet y = (x += 5)\n[x, y]", "[6, 6]"},
		{"let a = [1, 2]\na[1] += 10\na", "[1, 12]"},
		{`let h = {"n": 1}` + "\nh.n *= 5\nh", `{"n": 5}`},
		{`let h = {}` + "\nh[\"count\"] ??= 0\nh.count += 1\nh", `{"count": 1}`},
		{"let x = null\nx ??= 5\nx", "5"},
		{"let x = 0\nx ??= 5\nx", "0"},
		{`let h = {"a": 1}` + "\n[h.a ??= 9, h.b ??= 9, h]", `[1, 9, {"a": 1, "b": 9}]`},
		{"let a = [null, 2]\n[a[0] ??= 7, a[1] ??= 7, a]", "[7, 2, [7, 2]]"},
		{"let calls = 0\nfn idx() { calls++\n0 }\nlet a = [1]\na[idx()] += 1\n[a, calls]", "[[2], 1]"},
		{"let total = 0\nrepeat i in 1..4 { total += i }\ntotal", "10"},
		{"fn f() { let n = 0\nfn() { n += 2 } }\nlet g = f()\ng()\ng()", "4"},
		{"const c = 1\nc += 1", "ERROR at line 2, column 3: cannot reassign constant: c"},
		{"missing += 1", `ERROR at line 1, column 1: "missing" is not defined`},
		{`let x = "a"` + "\nx -= 1", `ERROR at line 2, column 3: cannot use "-" with a string and an integer`},
		{"let x = 1\nx ==2", "false"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
