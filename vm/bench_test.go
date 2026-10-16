package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/compiler"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
)

const benchFib = `fn fib(n) { if n < 2 { return n }
fib(n - 1) + fib(n - 2) }
fib(25)`

const benchLoop = `let total = 0
repeat i in 1..300000 {
    if i % 3 == 0 { total = total + i }
}
total`

func benchmarkVM(b *testing.B, source string) {
	program := parser.New(lexer.New(source)).ParseProgram()
	for i := 0; i < b.N; i++ {
		c := compiler.New()
		bytecode, _ := c.Compile(program)
		if result := New(c.Globals()).Run(bytecode); result.Type() == object.ERROR_OBJ {
			b.Fatal(result.Inspect())
		}
	}
}

func benchmarkEvaluator(b *testing.B, source string) {
	program := parser.New(lexer.New(source)).ParseProgram()
	for i := 0; i < b.N; i++ {
		if result := evaluator.Eval(program, object.NewEnvironment()); result.Type() == object.ERROR_OBJ {
			b.Fatal(result.Inspect())
		}
	}
}

func BenchmarkFibVM(b *testing.B)         { benchmarkVM(b, benchFib) }
func BenchmarkFibEvaluator(b *testing.B)  { benchmarkEvaluator(b, benchFib) }
func BenchmarkLoopVM(b *testing.B)        { benchmarkVM(b, benchLoop) }
func BenchmarkLoopEvaluator(b *testing.B) { benchmarkEvaluator(b, benchLoop) }
