package compiler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/parser"
)

// FuzzCompile checks that no source makes the lexer, the parser or the
// compiler panic: bad code must give errors.
func FuzzCompile(f *testing.F) {
	examples, _ := filepath.Glob("../examples/*.tg")
	tour, _ := filepath.Glob("../playground/tour/*.tg")
	tests, _ := filepath.Glob("../tests/*.tg")
	for _, file := range append(append(examples, tour...), tests...) {
		if source, err := os.ReadFile(file); err == nil {
			f.Add(string(source))
		}
	}
	for _, seed := range []string{
		"", "{", "fn(", "match x {", "repeat [a, b] in", "x |> ", `"${`, "0x", "let {a, ...b} = c",
		"a?.b?.c ?? d", "try { } catch e { throw e }", "(x) => {a: 1}", "1..", "[...]",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		p := parser.New(lexer.New(source))
		program := p.ParseProgram()
		p.Diagnostics()
		if len(p.Errors()) > 0 {
			return
		}
		_ = program.String()
		c := New()
		c.Compile(program)
		c.Problems()
	})
}
