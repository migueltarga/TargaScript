package parser

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/lexer"
)

func TestStructStatement(t *testing.T) {
	p := New(lexer.New("struct Point { x, y = 1\n fn len() { x } }"))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("errors: %v", p.Errors())
	}
	stmt, ok := program.Statements[0].(*ast.StructStatement)
	if !ok {
		t.Fatalf("got %T", program.Statements[0])
	}
	if stmt.Name.Value != "Point" || len(stmt.Fields) != 2 || stmt.Defaults[0] != nil || stmt.Defaults[1] == nil || len(stmt.Methods) != 1 {
		t.Errorf("got %s", stmt.String())
	}
	// struct is only a keyword before a name
	p = New(lexer.New("let struct = 1\nstruct + 1"))
	p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Errorf("errors: %v", p.Errors())
	}
}

func TestStructErrors(t *testing.T) {
	for input, message := range map[string]string{
		"struct P { x, x }":                  "x",
		"struct P { fn a() {}\n fn a() {} }": "a",
		"struct P { x":                       "}",
	} {
		p := New(lexer.New(input))
		p.ParseProgram()
		if len(p.Errors()) == 0 || !strings.Contains(strings.Join(p.Errors(), "\n"), message) {
			t.Errorf("%s: got %v", input, p.Errors())
		}
	}
}
