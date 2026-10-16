package parser

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/lexer"
)

func TestEnumStatement(t *testing.T) {
	p := New(lexer.New("enum Status { Active, Blocked\n Pending; }\nlet enum = 1"))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("errors: %v", p.Errors())
	}
	stmt, ok := program.Statements[0].(*ast.EnumStatement)
	if !ok || stmt.String() != "enum Status { Active, Blocked, Pending }" {
		t.Errorf("got %v", program.Statements[0])
	}
}

func TestEnumErrors(t *testing.T) {
	for input, message := range map[string]string{
		"enum S { A, A }":   "has A twice",
		"enum S { values }": "values is the method",
		"enum S { A":        "missing its closing }",
		"enum S { 1 }":      "expected the name of a value",
	} {
		p := New(lexer.New(input))
		p.ParseProgram()
		if !strings.Contains(strings.Join(p.Errors(), "\n"), message) {
			t.Errorf("%s: got %v", input, p.Errors())
		}
	}
}
