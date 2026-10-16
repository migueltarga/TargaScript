package compiler

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/code"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/parser"
)

// A call of a global loads it itself, unless its arguments could change it
func TestCallGlobal(t *testing.T) {
	tests := []struct {
		source string
		fused  int
	}{
		{"fn f(n) { n }\nf(1)", 1},
		{"fn f(n) { n }\nlet a = 1\nf(a - 1)", 1},
		{"fn f(n) { n }\nf(f(1))", 1},
		{"fn f(n) { n }\nf(if true { 1 } else { 2 })", 0},
		{"let f = null\nlet g = 1\nf(g = 2)", 0},
	}
	for _, tt := range tests {
		p := parser.New(lexer.New(tt.source))
		bytecode, err := New().Compile(p.ParseProgram())
		if err != nil {
			t.Fatal(err)
		}
		listing := code.WordsString(bytecode.Main.Code)
		if got := strings.Count(listing, "CALLGLOBAL"); got != tt.fused {
			t.Errorf("%q: fused %v, want %v:\n%s", tt.source, got, tt.fused, listing)
		}
	}
}
