package parser

import (
	"testing"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/lexer"
)

func TestNullLiteralExpression(t *testing.T) {
	p := New(lexer.New("let x = null"))
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*ast.LetStatement)
	if !ok {
		t.Fatalf("not a LetStatement. got=%T", program.Statements[0])
	}
	if _, ok := stmt.Value.(*ast.NullLiteral); !ok {
		t.Fatalf("value not *ast.NullLiteral. got=%T", stmt.Value)
	}
}
