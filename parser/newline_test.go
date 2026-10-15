package parser

import (
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
)

func TestNewlineEndsAmbiguousExpressions(t *testing.T) {
	tests := []struct {
		input      string
		statements []string
	}{
		{"let a = 5\n-3", []string{"let a = 5", "(-3)"}},
		{"let f = g\n(1 + 2)", []string{"let f = g", "(1 + 2)"}},
		{"let x = y\n[1, 2]", []string{"let x = y", "[1, 2]"}},
		{"let a = 5 -\n3", []string{"let a = (5 - 3)"}},
		{"let a = 5\n+ 3", []string{"let a = (5 + 3)"}},
		{"f(1)\n(2)", []string{"f(1)", "2"}},
		{"f(\n1,\n2\n)", []string{"f(1, 2)"}},
		{"let a = [\n1,\n-2\n]", []string{"let a = [1, (-2)]"}},
		{"let a = 5 - 3", []string{"let a = (5 - 3)"}},
		{"arr[0]", []string{"(arr[0])"}},
	}

	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != len(tt.statements) {
			t.Fatalf("%q: expected %d statements, got %d: %s",
				tt.input, len(tt.statements), len(program.Statements), program.String())
		}
		for i, stmt := range program.Statements {
			if stmt.String() != tt.statements[i] {
				t.Errorf("%q: statement %d is %q, want %q", tt.input, i, stmt.String(), tt.statements[i])
			}
		}
	}
}
