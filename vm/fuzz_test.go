package vm

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/migueltarga/TargaScript/compiler"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
)

// FuzzRun checks that no program makes the VM panic. MaxSteps ends the
// ones that would run forever.
func FuzzRun(f *testing.F) {
	examples, _ := filepath.Glob("../examples/*.tg")
	tests, _ := filepath.Glob("../tests/*.tg")
	for _, file := range append(examples, tests...) {
		if source, err := os.ReadFile(file); err == nil {
			f.Add(string(source))
		}
	}
	for _, seed := range []string{
		"repeat true {}", "fn f() { f() }\nf()", "let a = [1]; a[-5] = 1", `"ab" * 3`, "1 in 5",
		"[1, 2].map(x => x * 2) |> print", "match 5 { 1..9 => 1, _ => 2 }", "try { throw 1 } catch e { e }",
		"let {a, ...b} = {a: 1, c: 2}; b", "0xff + 1_000", "max([])", "range(1, 10, -1)",
	} {
		f.Add(seed)
	}

	evaluator.SetOutput(io.Discard)
	f.Cleanup(func() { evaluator.SetOutput(os.Stdout) })

	f.Fuzz(func(t *testing.T, source string) {
		p := parser.New(lexer.New(source))
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			return
		}
		c := compiler.New()
		bytecode, err := c.Compile(program)
		if err != nil {
			return
		}
		machine := New(c.Globals())
		machine.MaxSteps = 10000
		machine.Run(bytecode)
	})
}

func TestMaxSteps(t *testing.T) {
	p := parser.New(lexer.New("let n = 0\nrepeat true { n++ }"))
	c := compiler.New()
	bytecode, err := c.Compile(p.ParseProgram())
	if err != nil {
		t.Fatal(err)
	}
	machine := New(c.Globals())
	machine.MaxSteps = 100
	result := machine.Run(bytecode)
	if err, ok := result.(*object.Error); !ok || err.Message != "the program took more than 100 steps" {
		t.Errorf("got %s", result.Inspect())
	}
}

func TestMaxStepsCountsContinue(t *testing.T) {
	p := parser.New(lexer.New("repeat true { continue }"))
	c := compiler.New()
	bytecode, err := c.Compile(p.ParseProgram())
	if err != nil {
		t.Fatal(err)
	}
	machine := New(c.Globals())
	machine.MaxSteps = 100
	if err, ok := machine.Run(bytecode).(*object.Error); !ok || err.Message != "the program took more than 100 steps" {
		t.Errorf("got %v", err)
	}
}
