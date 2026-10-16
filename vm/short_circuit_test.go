package vm

import "testing"

// Values computed before ??, && and || keep their values when the right
// side changes the variables they came from
func TestShortCircuitKeepsValues(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"fn f() { let a = 1\nlet b = null\n[a, b ?? (a = 5), a] }\nf()", "[1, 5, 5]"},
		{"fn f() { let a = 1\nlet b = null\na + (b ?? (a = 5)) }\nf()", "6"},
		{"fn f() { let i = 0\nlet xs = [0, 0]\nxs[i] = null ?? (i = 1)\n[xs, i] }\nf()", "[[1, 0], 1]"},
		{"fn f() { let a = true\n[a, a && (a = false), a] }\nf()", "[true, false, false]"},
		{"fn f() { let a = false\n[a, a || (a = 3), a] }\nf()", "[false, 3, 3]"},
		{"fn f() { let a = 1\nlet o = null\n[a, o?.x.y, a] }\nf()", "[1, null, 1]"},
		{"fn f() { let a = 1\nlet o = {x: {y: 2}}\n[a, o?.x.y, a] }\nf()", "[1, 2, 1]"},
		{"fn f(xs) { let counts = {}\nrepeat w in xs { counts[w] = (counts[w] ?? 0) + 1 }\ncounts }\nf([\"a\", \"b\", \"a\"])", `{"a": 2, "b": 1}`},
		{"fn f() { let a = 1\nlet b = null\nlet g = () => a\n[a, b ?? (a = 7), g()] }\nf()", "[1, 7, 7]"},
		{"fn f() { let a = 2\n[a, null ?? a ?? (a = 9), a] }\nf()", "[2, 2, 2]"},
		{"fn f(m) { let k = \"x\"\n[k, m[k] ?? (k = \"y\"), k] }\nf({})", `["x", "y", "y"]`},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}
