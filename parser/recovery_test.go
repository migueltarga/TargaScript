package parser

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
)

func TestErrorRecovery(t *testing.T) {
	tests := []struct {
		input      string
		errors     []string // one expected substring per error, in order
		statements []string // statements that still parse
	}{
		{
			input:      "let = 5\nlet b = 2\nprint(b +)\nlet c = 3",
			errors:     []string{"line 1:", "line 3:"},
			statements: []string{"let b = 2", "let c = 3"},
		},
		{
			input:  `print("abc`,
			errors: []string{"this string is never closed"},
		},
		{
			input:      "fn f() {\n  let = 1\n  return 2\n}\nlet x = 1 @ 2\nlet y = 3",
			errors:     []string{"line 2:", `unexpected "@"`},
			statements: []string{"fn f() { return 2 }", "let x = 1", "let y = 3"},
		},
		{
			input:      "if true { let = }\nlet y = 1",
			errors:     []string{"line 1:"},
			statements: []string{"if(true", "let y = 1"},
		},
		{
			input:      "let 1a = 2\nlet ok = 1",
			errors:     []string{`invalid variable name "1a"`},
			statements: []string{"let ok = 1"},
		},
	}

	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		program := p.ParseProgram()
		errors := p.Errors()

		if len(errors) != len(tt.errors) {
			t.Errorf("%q: expected %d errors, got %d:\n%s",
				tt.input, len(tt.errors), len(errors), strings.Join(errors, "\n"))
			continue
		}
		for i, want := range tt.errors {
			if !strings.Contains(errors[i], want) {
				t.Errorf("%q: error %d is %q, want it to contain %q", tt.input, i, errors[i], want)
			}
		}

		if len(program.Statements) != len(tt.statements) {
			t.Errorf("%q: expected %d statements, got %d: %q",
				tt.input, len(tt.statements), len(program.Statements), program.String())
			continue
		}
		for i, want := range tt.statements {
			if !strings.HasPrefix(program.Statements[i].String(), want) {
				t.Errorf("%q: statement %d is %q, want prefix %q",
					tt.input, i, program.Statements[i].String(), want)
			}
		}
	}
}
