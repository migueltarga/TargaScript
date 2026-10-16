package parser

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
)

func TestPatterns(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let [a, b] = pair", "let [a, b] = pair"},
		{"const {name, age} = person", "const {name, age} = person"},
		{"let [single] = list", "let [single] = list"},
	}
	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			t.Errorf("%q: %v", tt.input, p.Errors())
			continue
		}
		if got := program.String(); got != tt.expected {
			t.Errorf("%q parsed as %q", tt.input, got)
		}
	}
}

func TestPatternErrors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let [] = x", `expected at least one name in []`},
		{"let [a b] = x", `expected ",", found "b"`},
		{"let {1} = x", `expected a name, found "1"`},
		{"let [a, b = x", `expected ",", found "="`},
	}
	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()
		errors := strings.Join(p.Errors(), "\n")
		if !strings.Contains(errors, tt.expected) {
			t.Errorf("%q: got %q, want it to contain %q", tt.input, errors, tt.expected)
		}
	}
}
