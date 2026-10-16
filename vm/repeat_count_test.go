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
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
