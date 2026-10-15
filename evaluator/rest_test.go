package evaluator

import "testing"

func TestArrayRest(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[1, 2, 3].rest()", "[2, 3]"},
		{"[1].rest()", "[]"},
		{"[].rest()", "[]"},
		{"[].rest().length", "0"},
		{"let a = [1, 2]\na.rest()\na", "[1, 2]"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}
}
