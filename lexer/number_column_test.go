package lexer

import (
	"testing"

	"github.com/migueltarga/TargaScript/token"
)

func TestColumnsAfterNumbers(t *testing.T) {
	l := New("x = 12 + 3.5, y")

	expected := []struct {
		literal string
		column  int
	}{
		{"x", 1}, {"=", 3}, {"12", 5}, {"+", 8}, {"3.5", 10}, {",", 13}, {"y", 15},
	}

	for _, tt := range expected {
		tok := l.NextToken()
		if tok.Literal != tt.literal || tok.Column != tt.column {
			t.Errorf("got %q at column %d, want %q at column %d", tok.Literal, tok.Column, tt.literal, tt.column)
		}
	}
	if tok := l.NextToken(); tok.Type != token.EOF {
		t.Errorf("expected EOF, got %q", tok.Literal)
	}
}
