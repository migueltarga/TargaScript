package parser

import (
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
)

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let x = 1 +\nlet y = 2", `line 1:12: expected a value after "+"`},
		{"let x = 1 +", `line 1:12: expected a value after "+"`},
		{"let x = 1 + )", `line 1:13: expected a value after "+", found ")"`},
		{"let = 5", `line 1:5: expected a name, found "="`},
		{"print(1, 2", `line 1:11: expected ")", but the file ended`},
		{"if x > 1 { print(x)", `line 1:10: this "{" is never closed, add a "}" to end the block`},
		{"fn f() {\n  return 1\n", `line 1:8: this "{" is never closed`},
		{"let a = [1, 2", `line 1:14: expected "]", but the file ended`},
		{"fn 5() {}", `line 1:4: expected "(", found "5"`},
		{"repeat x in [1] print(x)", `line 1:17: expected "{" to start the repeat body, found "print"`},
		{"load data", `line 1:6: expected a file name in quotes after "load", found "data"`},
		{"a.5", `line 1:3: expected a property or method name after ".", found "5"`},
		{"99999999999999999999", `line 1:1: 99999999999999999999 is too large for an integer`},
		{")", `line 1:1: unexpected ")"`},
		{"let s = \"abc", `line 1:9: this string is never closed, add a " at its end`},
	}

	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()

		errors := p.Errors()
		if len(errors) == 0 {
			t.Errorf("%q: expected an error", tt.input)
			continue
		}
		if len(errors[0]) < len(tt.expected) || errors[0][:len(tt.expected)] != tt.expected {
			t.Errorf("%q: got %q, want it to start with %q", tt.input, errors[0], tt.expected)
		}
	}
}

func TestDiagnosticsHavePositions(t *testing.T) {
	p := New(lexer.New("let a = 1\nlet b = * 2"))
	p.ParseProgram()

	diagnostics := p.Diagnostics()
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 error, got %v", p.Errors())
	}
	d := diagnostics[0]
	if d.Line != 2 || d.Column != 9 || d.Message != `expected a value after "=", found "*"` {
		t.Errorf("got %+v", d)
	}
}
