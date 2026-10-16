package parser

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
)

func TestUnterminatedBlockComment(t *testing.T) {
	p := New(lexer.New("let a = 1\n/* never closed"))
	p.ParseProgram()

	errors := strings.Join(p.Errors(), "\n")
	if !strings.Contains(errors, "line 2:1: this comment is never closed") {
		t.Errorf("expected an error for the unclosed comment, got:\n%s", errors)
	}
}
