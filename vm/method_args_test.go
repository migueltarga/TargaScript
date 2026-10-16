package vm

import "testing"

// Builtin methods read their arguments where they are on the VM's stack,
// so nothing they return can change when those registers do
func TestMethodArgumentsStayTheirs(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let xs = []\nrepeat i in 1..3 { xs.insert(i) }\nxs", "[1, 2, 3]"},
		{"let xs = []\nrepeat i in 1..3 { xs.insert([i, i * 2]) }\nxs", "[[1, 2], [2, 4], [3, 6]]"},
		{"let out = []\nrepeat s in [\"a,b\", \"c\"] { out.insert(s.split(\",\")) }\nout", `[["a", "b"], ["c"]]`},
		{"let xs = [1, 2, 3, 4]\nlet parts = []\nrepeat i in 0..2 { parts.insert(xs.slice(i, i + 2)) }\nparts", "[[1, 2], [2, 3], [3, 4]]"},
		{"let xs = [3, 1, 2]\n[xs.map((x, i) => x + i), xs.filter(x => x > 1), xs.join(\"-\")]", `[[3, 2, 4], [3, 2], "3-1-2"]`},
		{`"abc".replace("a", "x") + "abc".replace("b", "y")`, "xbcayc"},
		{"fn f(xs, x) { xs.insert(x)\nxs }\nf(f([], 1), 2)", "[1, 2]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}
