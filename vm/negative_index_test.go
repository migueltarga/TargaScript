package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestNegativeIndexes(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"[1, 2, 3][-1]", "3"},
		{"[1, 2, 3][-3]", "1"},
		{"[1, 2, 3][-4]", "null"},
		{"[][-1]", "null"},
		{`"héllo"[-1]`, "o"},
		{`"héllo"[-4]`, "é"},
		{`"abc"[-4]`, "null"},
		{"let a = [1, 2, 3]; a[-1] = 9; a", "[1, 2, 9]"},
		{"let a = [1, 2]; a[-1] += 5; a", "[1, 7]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestNegativeIndexOutOfBounds(t *testing.T) {
	err, ok := testEval("let a = [1]; a[-2] = 0").(*object.Error)
	if !ok || !strings.Contains(err.Message, "index out of bounds: -2") {
		t.Errorf("got %v", err)
	}
}
