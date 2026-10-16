package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

// Additions and subtractions of registers give the same results on every path
func TestAddAndSubtract(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let a = 2\nlet b = 3\n[a + b, a - b]", "[5, -1]"},
		{"let a = 2305843009213693951\nlet b = 1\na + b", "2305843009213693952"},
		{"let a = 1\nlet b = 0.5\n[a + b, a - b]", "[1.5, 0.5]"},
		{`let a = "x"` + "\nlet b = \"y\"\na + b", "xy"},
		{"let a = [1]\nlet b = [2]\na + b", "[1, 2]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
	for _, input := range []string{"let a = 9223372036854775807\nlet b = 1\na + b", "let a = -9223372036854775807\nlet b = 2\na - b"} {
		if result, ok := testEval(input).(*object.Error); !ok || !strings.Contains(result.Message, "overflow") {
			t.Errorf("%s: got %v, want an overflow", input, result)
		}
	}
	if !isError(testEval("let a = 1\nlet b = \"x\"\na - b")) {
		t.Error("1 - \"x\" should fail")
	}
}
