package lexer

import (
	"testing"

	"github.com/migueltarga/TargaScript/token"
)

func TestBlockComments(t *testing.T) {
	input := `let a = /* inline */ 1
/*
  multi
  line * / comment
*/
let b = 2 /**/ / 3
/* unterminated`

	expected := []struct {
		expectedType    token.TokenType
		expectedLiteral string
		line            int
	}{
		{token.LET, "let", 1},
		{token.IDENT, "a", 1},
		{token.ASSIGN, "=", 1},
		{token.INT, "1", 1},
		{token.LET, "let", 6},
		{token.IDENT, "b", 6},
		{token.ASSIGN, "=", 6},
		{token.INT, "2", 6},
		{token.SLASH, "/", 6},
		{token.INT, "3", 6},
		{token.ILLEGAL, "/*", 7},
		{token.EOF, "", 7},
	}

	l := New(input)
	for i, tt := range expected {
		tok := l.NextToken()
		if tok.Type != tt.expectedType || tok.Literal != tt.expectedLiteral || tok.Line != tt.line {
			t.Fatalf("tests[%d] - got %s %q line %d, want %s %q line %d",
				i, tok.Type, tok.Literal, tok.Line, tt.expectedType, tt.expectedLiteral, tt.line)
		}
	}
}
