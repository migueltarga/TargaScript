package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestNumberLiterals(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"1_000_000", "1000000"},
		{"0xff", "255"},
		{"0XFF", "255"},
		{"0b1010", "10"},
		{"0o17", "15"},
		{"0xFF_FF", "65535"},
		{"010", "10"},
		{"1_000.5", "1000.5"},
		{"1_0e2", "1000"},
		{"0x10 + 1", "17"},
		{"let a = [1, 2]; a[0x1]", "2"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestBadNumberLiterals(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"0x", "0x is not a valid number"},
		{"0b12", "0b12 is not a valid number"},
		{"0xFFFFFFFFFFFFFFFFFF", "too large"},
	}
	for _, tt := range tests {
		got := testEval(tt.input)
		err, ok := got.(*object.Error)
		if !ok || !strings.Contains(err.Message, tt.expected) {
			t.Errorf("%s: got %v, want an error with %q", tt.input, got, tt.expected)
		}
	}
}
