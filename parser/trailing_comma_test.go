package parser

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
)

func TestTrailingCommas(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[1, 2,]", "[1, 2]"},
		{"[\n  1,\n  2,\n]", "[1, 2]"},
		{"f(1, 2,)", "f(1, 2)"},
		{"obj.m(1,)", "obj.m(1)"},
		{"fn(a, b,) { a }", "fn(a, b) { a }"},
		{"print(1, 2,)", "print(1, 2)"},
		{"[]", "[]"},
		{"f()", "f()"},
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

func TestInvalidListSyntax(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[1,, 2]", "no prefix parse function for ,"},
		{"[,]", "no prefix parse function for ,"},
		{"f(1 2)", "expected next token to be )"},
		{"fn(1 + 2) {}", `expected parameter name, got "1"`},
		{"fn(a,, b) {}", `expected parameter name, got ","`},
	}

	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()
		errors := strings.Join(p.Errors(), "\n")
		if !strings.Contains(errors, tt.expected) {
			t.Errorf("%q: expected an error containing %q, got:\n%s", tt.input, tt.expected, errors)
		}
	}
}
