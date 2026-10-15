package parser

import (
	"testing"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/lexer"
)

func TestIntegerLiteralLeadingZeros(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"010", 10},
		{"08", 8},
		{"007", 7},
		{"0", 0},
	}

	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		literal, ok := stmt.Expression.(*ast.IntegerLiteral)
		if !ok {
			t.Fatalf("%q: exp not *ast.IntegerLiteral. got=%T", tt.input, stmt.Expression)
		}
		if literal.Value != tt.expected {
			t.Errorf("%q: got=%d, want=%d", tt.input, literal.Value, tt.expected)
		}
	}
}
