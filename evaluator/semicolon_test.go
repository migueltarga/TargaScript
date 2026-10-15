package evaluator

import (
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/parser"
)

func TestSemicolonSeparatedStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"let a = 1; let b = 2; a + b", 3},
		{"let i = 0; [1][i]", 1},
		{"fn f(x) { let y = x * 2; return y; }; f(4)", 8},
		{"let n = 0; repeat i in 3 { n = n + i; }; n", 6},
		{";;let a = 1;; a", 1},
	}

	for _, tt := range tests {
		p := parser.New(lexer.New(tt.input))
		p.ParseProgram()
		if len(p.Errors()) > 0 {
			t.Fatalf("%q: parser errors: %v", tt.input, p.Errors())
		}
		testIntegerObject(t, testEval(tt.input), tt.expected, tt.input)
	}
}
