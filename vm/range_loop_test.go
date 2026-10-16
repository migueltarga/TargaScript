package vm

import "testing"

// Loops that take the next value at their end still see every value once
func TestLoopsAtTheirEnd(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"let s = 0\nrepeat i in 1..4 { s += i }\ns", 10},
		{"let s = 0\nrepeat i in 4..1 { s += i }\ns", 0},
		{"let s = 0\nrepeat i in 9223372036854775806..9223372036854775807 { s += 1 }\ns", 2},
		{"let s = 0\nrepeat i in 1..10 { if i % 2 == 0 { continue }\ns += i }\ns", 25},
		{"let s = 0\nrepeat i in 1..10 { if i > 3 { break }\ns += i }\ns", 6},
		{"let s = 0\nrepeat x in [5, 6, 7] { s += x }\ns", 18},
		{"let s = 0\nrepeat k in {\"a\": 1, \"bc\": 2} { s += k.length }\ns", 3},
		{"fn f() { let s = 0\nrepeat i in 1..3 { s += i }\ns }\nf()", 6},
		{"let fs = []\nrepeat i in 1..3 { fs.insert(() => i) }\nfs[0]() + fs[2]()", 4},
	}
	for _, tt := range tests {
		testIntegerObject(t, testEval(tt.input), tt.expected, tt.input)
	}
}
