package parser

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/lexer"
)

func TestTemplateLiteralParsing(t *testing.T) {
	p := New(lexer.New(`"a ${x + 1} b ${f(y)}"`))
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	tmpl, ok := stmt.Expression.(*ast.TemplateLiteral)
	if !ok {
		t.Fatalf("exp not *ast.TemplateLiteral. got=%T", stmt.Expression)
	}

	expected := []string{`"a "`, "(x + 1)", `" b "`, "f(y)"}
	if len(tmpl.Parts) != len(expected) {
		t.Fatalf("wrong number of parts. got=%d", len(tmpl.Parts))
	}
	for i, want := range expected {
		if got := tmpl.Parts[i].String(); got != want {
			t.Errorf("part %d: got=%s, want=%s", i, got, want)
		}
	}

	if got := tmpl.String(); got != `"a ${(x + 1)} b ${f(y)}"` {
		t.Errorf("String() got=%s", got)
	}
}

func TestPlainStringsStayStrings(t *testing.T) {
	for _, input := range []string{`"no interpolation"`, `"price: $5"`, `"escaped: \${x}"`} {
		p := New(lexer.New(input))
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		if _, ok := stmt.Expression.(*ast.StringLiteral); !ok {
			t.Errorf("%s: expected *ast.StringLiteral, got=%T", input, stmt.Expression)
		}
	}
}

func TestTemplateLiteralErrors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"${}"`, "empty ${} in string"},
		{`"${1 +}"`, `line 1:1: in ${1 +}: expected a value after "+"`},
		{`"${1 2}"`, `in ${1 2}: unexpected "2"`},
		{`"${1"`, "this string is never closed"},
		{`"${"a}"`, "this string is never closed"},
	}

	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()
		errors := strings.Join(p.Errors(), "\n")
		if !strings.Contains(errors, tt.expected) {
			t.Errorf("%s: expected an error containing %q, got:\n%s", tt.input, tt.expected, errors)
		}
	}
}
