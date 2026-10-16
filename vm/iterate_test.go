package vm

import "testing"

// Loops see the collection as it was when they started
func TestIterating(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let xs = [1, 2]\nlet seen = []\nrepeat x in xs { xs.insert(x)\nseen.insert(x) }\n[seen, xs]", "[[1, 2], [1, 2, 1, 2]]"},
		{"let o = {a: 1}\nlet seen = []\nrepeat k in o { o[\"b\"] = 2\nseen.insert(k) }\n[seen, o.keys()]", `[["a"], ["a", "b"]]`},
		{"let out = []\nrepeat i, x in [\"a\", \"b\"] { out.insert(\"${i}${x}\") }\nout", `["0a", "1b"]`},
		{"let out = []\nrepeat k, v in {x: 1, y: 2} { out.insert(\"${k}${v}\") }\nout", `["x1", "y2"]`},
		{"let out = []\nrepeat i, c in \"hé\" { out.insert(\"${i}${c}\") }\nout", `["0h", "1é"]`},
		{"let out = []\nrepeat c in \"hé\" { out.insert(c) }\nout", `["h", "é"]`},
		{"let n = 0\nrepeat x in [] { n += 1 }\nrepeat k in {} { n += 1 }\nn", "0"},
		{"fn f(xs) { let s = 0\nrepeat x in xs { if x > 2 { break }\ns += x }\ns }\nf([1, 2, 3, 4])", "3"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}
