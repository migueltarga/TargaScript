package lexer

import (
	"testing"

	"github.com/migueltarga/TargaScript/token"
)

func TestSemicolon(t *testing.T) {
	expected := []token.TokenType{
		token.LET, token.IDENT, token.ASSIGN, token.INT, token.SEMICOLON,
		token.IDENT, token.SEMICOLON, token.EOF,
	}

	l := New("let a = 1; a;")
	for i, tt := range expected {
		tok := l.NextToken()
		if tok.Type != tt {
			t.Fatalf("tests[%d] - got %s %q, want %s", i, tok.Type, tok.Literal, tt)
		}
	}
}
