package parser

import (
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
)

func TestRangePrecedence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1..5", "(1 .. 5)"},
		{"a..b + 1", "(a .. (b + 1))"},
		{"a * 2..b", "((a * 2) .. b)"},
		{"x == 1..3", "(x == (1 .. 3))"},
		{"a..b < c", "((a .. b) < c)"},
	}

	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		program := p.ParseProgram()
		checkParserErrors(t, p)
		if got := program.String(); got != tt.expected {
			t.Errorf("%q: got=%q, want=%q", tt.input, got, tt.expected)
		}
	}
}
