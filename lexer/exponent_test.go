package lexer

import (
	"testing"

	"github.com/migueltarga/TargaScript/token"
)

func TestNumberExponents(t *testing.T) {
	input := `1e3 2.5E-4 7e+2 1.5e10 3e 4ex`

	expected := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.FLOAT, "1e3"},
		{token.FLOAT, "2.5E-4"},
		{token.FLOAT, "7e+2"},
		{token.FLOAT, "1.5e10"},
		{token.ILLEGAL, "3e"},
		{token.ILLEGAL, "4ex"},
		{token.EOF, ""},
	}

	l := New(input)
	for i, tt := range expected {
		tok := l.NextToken()
		if tok.Type != tt.expectedType || tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - got %s %q, want %s %q",
				i, tok.Type, tok.Literal, tt.expectedType, tt.expectedLiteral)
		}
	}
}
