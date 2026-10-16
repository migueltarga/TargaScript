package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestArrowRestParameters(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let all = (...xs) => xs; all(1, 2, 3)", "[1, 2, 3]"},
		{"let all = (...xs) => xs; all()", "[]"},
		{"let tail = (first, ...rest) => [first, rest]; tail(1, 2, 3)", "[1, [2, 3]]"},
		{"let sum = (...ns) => ns.sum(); sum(...[1, 2], 3)", "6"},
		{"[1, 2].map((...args) => args)", "[[1, 0], [2, 1]]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}

	err, ok := testEval("let bad = (...xs, y) => 1").(*object.Error)
	if !ok || !strings.Contains(err.Message, "...name must be the last parameter") {
		t.Errorf("rest before another parameter: got %v", err)
	}
}
