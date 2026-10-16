package evaluator

import "testing"

func TestEditDistance(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"length", "lenght", 1},
		{"abc", "abc", 0},
		{"", "abc", 3},
		{"kitten", "sitting", 3},
		{"filter", "fliter", 1},
	}
	for _, tt := range tests {
		if got := editDistance(tt.a, tt.b); got != tt.expected {
			t.Errorf("editDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
		}
	}
}
