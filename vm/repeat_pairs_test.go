package vm

import "testing"

func TestRepeatKeyValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`let out = []` + "\nrepeat i, name in [\"a\", \"b\"] { out.insert(\"${i}:${name}\") }\nout", `["0:a", "1:b"]`},
		{`let out = []` + "\nrepeat key, value in {\"x\": 1, \"y\": 2} { out.insert([key, value]) }\nout", `[["x", 1], ["y", 2]]`},
		{`let out = []` + "\nrepeat i, c in \"hé\" { out.insert([i, c]) }\nout", `[[0, "h"], [1, "é"]]`},
		{`let out = []` + "\nrepeat i, n in 5..7 { out.insert([i, n]) }\nout", "[[0, 5], [1, 6], [2, 7]]"},
		{`let out = []` + "\nrepeat i, n in 3 { out.insert([i, n]) }\nout", "[[0, 1], [1, 2], [2, 3]]"},
		{`let out = []` + "\nrepeat k in {\"x\": 1} { out.insert(k) }\nout", `["x"]`},
		{"let total = 0\nrepeat i, x in [10, 20, 30] { if i == 1 { continue }\ntotal = total + x }\ntotal", "40"},
		{"let fs = []\nrepeat i, x in [\"a\", \"b\"] { fs.insert(fn() { "${i}${x}" }) }\nfs.map(fn(f) { f() })", `["0a", "1b"]`},
		{"repeat i, x in 5.5 { }", "ERROR at line 1, column 1: cannot iterate over FLOAT"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
