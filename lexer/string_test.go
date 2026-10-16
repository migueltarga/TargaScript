package lexer

import (
	"testing"

	"github.com/migueltarga/TargaScript/token"
)

func TestUnterminatedString(t *testing.T) {
	l := New(`print("abc`)

	expected := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.IDENT, "print"},
		{token.LPAREN, "("},
		{token.ILLEGAL, `"abc`},
		{token.EOF, ""},
	}

	for i, tt := range expected {
		tok := l.NextToken()
		if tok.Type != tt.expectedType || tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - got %s %q, want %s %q",
				i, tok.Type, tok.Literal, tt.expectedType, tt.expectedLiteral)
		}
	}
}
