package evaluator

import "testing"

func TestArrayInsertModifiesArray(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let a = [1, 2]\na.insert(3)\na", "[1, 2, 3]"},
		{"let a = []\na = a.insert(1)\na", "[1]"},
		{"let a = [1]\nlet b = a\nb.insert(2)\na", "[1, 2]"},
		{"let a = [1]\na.insert(2).insert(3)\na", "[1, 2, 3]"},
		{"let h = {\"items\": []}\nh.items.insert(\"x\")\nh.items", "[x]"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}
}
