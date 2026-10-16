package compiler

import (
	"testing"

	"github.com/migueltarga/TargaScript/code"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
)

// A branch that ends a function returns its value from where it is
func TestReturnsInPlace(t *testing.T) {
	p := parser.New(lexer.New("fn f(n) { if n < 2 { n } else { n + 1 } }"))
	bytecode, err := New().Compile(p.ParseProgram())
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range bytecode.Constants {
		f, ok := k.(*object.CompiledFunction)
		if !ok {
			continue
		}
		if w := f.Code[1]; w.Op() != code.RReturn || w.A() != 0 {
			t.Fatalf("the first branch should return n, got:\n%s", code.WordsString(f.Code))
		}
		return
	}
	t.Fatal("no function compiled")
}
