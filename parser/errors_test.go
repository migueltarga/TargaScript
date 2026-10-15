package parser

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
)

func TestIllegalTokenErrors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`let s = "abc`, "unterminated string"},
		{`let a = 1 @ 2`, `unexpected "@"`},
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
