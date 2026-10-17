package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

// Bitwise operators work on integers, with Go's precedence: & and the
// shifts like *, and | and ^ like +
func TestBitwise(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"[12 & 10, 12 | 10, 12 ^ 10, ~5, ~-1]", "[8, 14, 6, -6, 0]"},
		{"[1 << 10, -16 >> 2, 1 << 63, 1 << 64, -1 >> 70, 5 >> 64]", "[1024, -4, -9223372036854775808, 0, -1, 0]"},
		{"let a = 12\nlet b = 10\n[a & b, a | b, a << 2, a >> b]", "[8, 14, 48, 0]"},
		{"[1 + 2 & 3, 1 | 2 + 4, 1 << 2 * 3, 2 * 3 << 1, 1 < 2 << 1]", "[3, 7, 12, 12, true]"},
		{"let x = 5\nx &= 3\nx |= 8\nx <<= 2\nx >>= 1\nx ^= 1\nx", "19"},
		{"fn rotl(v, n) { ((v << n) | (v >> (32 - n))) & 0xFFFFFFFF }\nrotl(0x80000001, 1)", "3"},
		{"let a = [1]\na[0] |= 6\na", "[7]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestBitwiseErrors(t *testing.T) {
	tests := []struct{ input, message string }{
		{"1 << -1", "cannot shift by a negative number: -1"},
		{"let n = -1\n1 >> n", "cannot shift by a negative number: -1"},
		{"1.5 & 1", `cannot use "&"`},
		{`"a" | 1`, `cannot use "|"`},
	}
	for _, tt := range tests {
		result, ok := testEval(tt.input).(*object.Error)
		if !ok || !strings.Contains(result.Message, tt.message) {
			t.Errorf("%s: got %v, want %q", tt.input, result, tt.message)
		}
	}
}
