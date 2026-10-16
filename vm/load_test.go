package vm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/migueltarga/TargaScript/compiler"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
)

func writeModules(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func runInDir(t *testing.T, dir, input string) object.Object {
	t.Helper()
	program := parser.New(lexer.New(input)).ParseProgram()
	c := compiler.New()
	c.Dir = dir
	bytecode, err := c.Compile(program)
	if err != nil {
		t.Fatal(err)
	}
	return New(c.Globals()).Run(bytecode)
}

func TestLoad(t *testing.T) {
	dir := writeModules(t, map[string]string{
		"lib/math.tg": "fn square(x) { x * x }\nconst PI = 3\nlet loads = 0",
		"lib/geo.tg":  "load \"math.tg\"\nfn area(r) { PI * square(r) }\nfn later() {\n  load \"math.tg\" as m\n  return m.square(4)\n}",
		"counter.tg":  "let count = 0\nfn inc() { count = count + 1\nreturn count }",
		"math.tg":     "const PI = 3",
		"a.tg":        "load \"b.tg\"",
		"b.tg":        "load \"a.tg\"",
		"broken.tg":   "let x = 1\nprint(missing)",
		"syntax.tg":   "let = 1",
		"apply.tg":    "fn twice(f, x) { f(f(x)) }",
	})

	tests := []string{
		"load \"lib/math.tg\"\nsquare(3)",
		"load \"lib/math.tg\"\nPI",
		"load \"lib/math.tg\" as m\nm.square(5) + m.PI",
		"load \"lib/math.tg\" as m\nm.keys()",
		"load \"lib/geo.tg\" as geo\ngeo.area(2)",
		"load \"lib/geo.tg\" as geo\ngeo.later()",
		"load \"counter.tg\" as a\nload \"counter.tg\" as b\na.inc()\nb.inc()",
		"load \"lib/math.tg\";\nsquare(2)",
		"fn f() { load \"lib/math.tg\"\nsquare(6) }\nf()",
		"load \"apply.tg\"\ntwice(fn(n) { n * 10 }, 2)",
		"load \"math.tg\"\nPI = 4",
		"const PI = 1\nload \"math.tg\"",
		"load \"nope.tg\"",
		"load \"a.tg\"",
		"load \"broken.tg\"",
		"load \"syntax.tg\"",
		"const m = 1\nload \"math.tg\" as m",
	}

	for _, input := range tests {
		evaluator.SetMainDir(dir)
		evaluator.ResetModules()
		program := parser.New(lexer.New(input)).ParseProgram()
		want := describe(evaluator.Eval(program, object.NewEnvironment()))
		got := describe(runInDir(t, dir, input))
		if got != want {
			t.Errorf("%q\n  vm:        %s\n  evaluator: %s", input, got, want)
		}
	}
	evaluator.SetMainDir("")
}
