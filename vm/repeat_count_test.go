package vm

import "testing"

func TestRepeatCount(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let n = 0\nrepeat 3 { n++ }\nn", "3"},
		{"let n = 0\nrepeat 0 { n++ }\nn", "0"},
		{"let n = 0\nrepeat 5 { if n == 2 { break }\nn++ }\nn", "2"},
		{"let n = 3\nlet steps = 0\nrepeat n { n--\nsteps++ }\nsteps", "3"},
		{"let i = 0\nrepeat i < 4 { i++ }\ni", "4"},
		// A number from a name, a property or arithmetic counts too, taken
		// once at the start
		{"let n = 3\nlet steps = 0\nrepeat n { steps++ }\nsteps", "3"},
		{"let o = {k: 2}\nlet steps = 0\nrepeat o.k { steps++ }\nsteps", "2"},
		{"let n = 3\nlet steps = 0\nrepeat n * 2 - 1 { steps++ }\nsteps", "5"},
		{"let n = 4\nlet steps = 0\nrepeat n { n = 1\nsteps++ }\nsteps", "4"},
		{"let n = -1\nlet steps = 0\nrepeat n { steps++ }\nsteps", "0"},
		{"let n = 3\nlet seen = []\nrepeat n { seen.insert(n)\nif seen.length == 1 { continue }\nn++ }\nseen", "[3, 3, 4]"},
		{"fn f() { let t = 2\nlet steps = 0\nrepeat t { steps++ }\nsteps }\nf()", "2"},
		// Anything else is a condition, taken again each round, and so is
		// a call
		{"let q = [1, 5]\nlet n = 0\nrepeat q.pop() { n++ }\nn", "2"},
		{"let xs = [1, 2, 3]\nrepeat xs { xs.pop() }\nxs", "[]"},
		{"let more = true\nlet i = 0\nrepeat more { i++\nmore = i < 3 }\ni", "3"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
