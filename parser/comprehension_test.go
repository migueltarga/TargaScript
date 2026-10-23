package parser

import (
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
)

func TestListComprehension(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[x * 2 repeat x in xs]", "[(x * 2) repeat x in xs]"},
		{"[x repeat x in xs if x > 1]", "[x repeat x in xs if (x > 1)]"},
		{"[f(x) repeat k, v in o repeat [a, b] in v]", "[f(x) repeat k, v in o repeat [a, b] in v]"},
		{"[x repeat x in 1..3]", "[x repeat x in (1 .. 3)]"},
		{"[1, 2]", "[1, 2]"},
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
