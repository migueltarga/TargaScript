package lexer

import (
	"testing"

	"github.com/migueltarga/TargaScript/token"
)

func TestTokenPositions(t *testing.T) {
	input := "let x = 5\n  print(x)\n\ty"

	tests := []struct {
		expectedType   token.TokenType
		expectedLine   int
		expectedColumn int
	}{
		{token.LET, 1, 1},
		{token.IDENT, 1, 5},
		{token.ASSIGN, 1, 7},
		{token.INT, 1, 9},
		{token.PRINT, 2, 3},
		{token.LPAREN, 2, 8},
		{token.IDENT, 2, 9},
		{token.RPAREN, 2, 10},
		{token.IDENT, 3, 2},
	}

	l := New(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - type wrong. expected=%q, got=%q", i, tt.expectedType, tok.Type)
		}
		if tok.Line != tt.expectedLine || tok.Column != tt.expectedColumn {
			t.Errorf("tests[%d] - %q at %d:%d, want %d:%d",
				i, tok.Literal, tok.Line, tok.Column, tt.expectedLine, tt.expectedColumn)
		}
	}
}
