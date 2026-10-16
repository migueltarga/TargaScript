package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/compiler"
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

const benchObjects = `let people = []
repeat i in 1..20000 {
    people.insert({"name": "p" + i, "age": i % 90})
}
let total = 0
repeat p in people {
    if p.age > 30 { total = total + p.age * 2 }
}
total`

const benchMethods = `let numbers = (1..50000).map(fn(n) { n * 3 })
numbers.filter(fn(n) { n % 2 == 0 }).map(fn(n) { n / 3 }).reduce(fn(a, b) { a + b }, 0)`

func BenchmarkFibVM(b *testing.B)     { benchmarkVM(b, benchFib) }
func BenchmarkLoopVM(b *testing.B)    { benchmarkVM(b, benchLoop) }
func BenchmarkObjectsVM(b *testing.B) { benchmarkVM(b, benchObjects) }
func BenchmarkMethodsVM(b *testing.B) { benchmarkVM(b, benchMethods) }
