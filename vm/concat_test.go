package vm

import "testing"

func TestArrayConcat(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"[1, 2] + [3]", "[1, 2, 3]"},
		{"[] + []", "[]"},
		{"let a = [1]; let b = a + [2]; a", "[1]"},
		{"let a = [1]; a += [2, 3]; a", "[1, 2, 3]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}
