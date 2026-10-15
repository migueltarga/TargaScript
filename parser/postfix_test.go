package parser

import (
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
)

func TestPostfixParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"x++", "(x++)"},
		{"x--", "(x--)"},
		{"obj.count++", "(obj.count++)"},
		{"a[i]--", "((a[i])--)"},
		{"x++ + 1", "((x++) + 1)"},
		{"-x++", "(-(x++))"},
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
