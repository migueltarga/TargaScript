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
	if !strings.Contains(errors, "line 2:1: unterminated comment") {
		t.Errorf("expected an unterminated comment error, got:\n%s", errors)
	}
}
