package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/compiler"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
)

// A loop that takes too many steps fails at the loop, whatever its condition
func TestLoopStepsPointAtTheLoop(t *testing.T) {
	for _, input := range []string{
		"let r = true\n  repeat r {}",
		"fn f() { let r = true\n  repeat r {} }\nf()",
		"fn f() { let s = 0\n  repeat i in 1..100000 { s += i } }\nf()",
	} {
		p := parser.New(lexer.New(input))
		c := compiler.New()
		bytecode, err := c.Compile(p.ParseProgram())
		if err != nil {
			t.Fatal(err)
		}
		machine := New(c.Globals())
		machine.MaxSteps = 100
		result, ok := machine.Run(bytecode).(*object.Error)
		if !ok || result.Line != 2 || result.Column != 3 {
			t.Errorf("%q: got %#v", input, result)
		}
	}
}
