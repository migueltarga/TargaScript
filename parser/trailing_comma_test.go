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
		{"[1,, 2]", `expected a value after ",", found ","`},
		{"[,]", `expected a value after "[", found ","`},
		{"f(1 2)", `expected ")", found "2"`},
		{"fn(1 + 2) {}", `expected a parameter name, found "1"`},
		{"fn(a,, b) {}", `expected a parameter name, found ","`},
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
