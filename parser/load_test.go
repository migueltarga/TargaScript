package parser

import (
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
)

func TestLoadDoesNotSkipNextStatement(t *testing.T) {
	tests := []struct {
		input      string
		statements []string
	}{
		{"load \"a.tg\"\nprint(1)", []string{`load "a.tg"`, "print(1)"}},
		{"load \"a.tg\" as a\nprint(1)", []string{`load "a.tg" as a`, "print(1)"}},
		{"load \"a.tg\"; let x = 1", []string{`load "a.tg"`, "let x = 1"}},
	}

	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != len(tt.statements) {
			t.Fatalf("%q: expected %d statements, got %d", tt.input, len(tt.statements), len(program.Statements))
		}
		for i, want := range tt.statements {
			if got := program.Statements[i].String(); got != want {
				t.Errorf("%q: statement %d is %q, want %q", tt.input, i, got, want)
			}
		}
	}
}
