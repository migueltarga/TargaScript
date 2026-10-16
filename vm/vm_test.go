package vm

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/migueltarga/TargaScript/compiler"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
)

// result is what running a program produced: its value or error, and its
// output.
type result struct {
	value  string
	output string
}

// sourceDir is the directory load paths are relative to.
var sourceDir string

func describe(obj object.Object) string {
	if err, ok := obj.(*object.Error); ok {
		return "ERROR " + err.Inspect()
	}
	if obj == nil {
		return "<nil>"
	}
	return string(obj.Type()) + " " + obj.Inspect()
}

func parse(t *testing.T, input string) *parser.Parser {
	return parser.New(lexer.New(input))
}

func runVM(t *testing.T, input string) result {
	t.Helper()
	p := parse(t, input)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("%q: parser errors: %v", input, p.Errors())
	}

	var out bytes.Buffer
	evaluator.SetOutput(&out)
	defer evaluator.SetOutput(os.Stdout)

	c := compiler.New()
	c.Dir = sourceDir
	bytecode, err := c.Compile(program)
	if err != nil {
		t.Fatalf("%q: compiler error: %s", input, err)
	}
	machine := New(c.Globals())
	return result{describe(machine.Run(bytecode)), out.String()}
}

// update rewrites the expected results in testdata with what the VM
// produces now: go test ./vm -update. They were first recorded from the
// tree-walking evaluator the VM replaced.
var update = flag.Bool("update", false, "rewrite the expected results in testdata")

var (
	goldenOnce sync.Once
	golden     map[string]result
)

func loadGolden(t *testing.T) map[string]result {
	goldenOnce.Do(func() {
		golden = map[string]result{}
		data, err := os.ReadFile("testdata/golden.json")
		if err != nil {
			t.Fatal(err)
		}
		raw := map[string]struct{ Value, Output string }{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatal(err)
		}
		for input, r := range raw {
			golden[input] = result{r.Value, r.Output}
		}
	})
	return golden
}

func TestMain(m *testing.M) {
	flag.Parse()
	code := m.Run()
	if *update && golden != nil {
		raw := map[string]map[string]string{}
		for input, r := range golden {
			raw[input] = map[string]string{"value": r.value, "output": r.output}
		}
		data, _ := json.MarshalIndent(raw, "", " ")
		os.WriteFile("testdata/golden.json", append(data, '\n'), 0o644)
	}
	os.Exit(code)
}

// expectGolden checks that each program gives the value, error and output
// recorded in testdata/golden.json.
func expectGolden(t *testing.T, inputs []string) {
	t.Helper()
	expected := loadGolden(t)
	for _, input := range inputs {
		got := runVM(t, input)
		want, ok := expected[input]
		if *update {
			expected[input] = got
			continue
		}
		if !ok {
			t.Errorf("%q: no expected result, run go test ./vm -update", input)
			continue
		}
		if got != want {
			t.Errorf("%q\n  got:  %s | output %q\n  want: %s | output %q",
				input, got.value, got.output, want.value, want.output)
		}
	}
}

func TestLiteralsAndOperators(t *testing.T) {
	expectGolden(t, []string{
		"1", "1.5", `"hi"`, "true", "false", "null",
		"1 + 2 * 3", "(1 + 2) * 3", "10 / 3", "10 % 3", "7.5 / 2", "1 / 0", "1.0 / 0",
		"5 ^ 3", "2 < 3", "2 >= 3", "1 == 1.0", `"a" == "a"`, `"a" + 1`, `"a" - 1`,
		"-5", "!true", "!0", `-"a"`, "9223372036854775807 + 1", "1 + true",
		"1..5", "5..1", `"a"..1`,
		"true && 0", "0 || 5", "null || false", "1 && 2 && 3",
		"[1, 2, [3]]", `{"a": 1, b: [2]}`, `{[1]: 2}`, `"x${1 + 1}y${[1]}"`,
		"[1, 2, 3][1]", "[1][5]", `{"a": 1}["a"]`, `{"a": 1}.a`, `"abc"[1]`, "5[0]",
	})
}

func TestVariablesAndScopes(t *testing.T) {
	expectGolden(t, []string{
		"let a = 1\na", "let a = 1\nlet a = 2\na", "const c = 1\nc = 2", "const c = 1\nconst c = 2",
		"const c = 1\nlet c = 2", "x", "x = 1", "let a = 1\na = 5\na",
		"let a = 1\nif true { let a = 2\na } ", "let a = 1\nif true { a = 2 }\na",
		"if true { let inner = 1 }\ninner", "let max = 3\nmax", "max(1, 2)",
		"if false { 1 }", "if 1 > 2 { 1 } else { 2 }", "if true { }",
		"let x = 1\nx++\nx", "let x = 1.5\nx--", `let s = "a"` + "\ns++",
		"let a = [1, 2]\na[0] = 5\na", "let a = [1]\na[3] = 1", `let h = {}` + "\nh.x = 1\nh.x++\nh",
		`let h = {"n": 1}` + "\nh[\"n\"]++\nh", "let a = [1]\na[0]++\na",
		"totl", "let total = 1\ntotl",
	})
}

func TestFunctionsAndClosures(t *testing.T) {
	expectGolden(t, []string{
		"fn add(a, b) { a + b }\nadd(1, 2)", "let f = fn(x) { x * 2 }\nf(4)",
		"fn f(a, b) { b }\nf(1)", "fn f(a) { a }\nf(1, 2, 3)", "fn f() { return }\nf()",
		"fn f() { }\nf()", "fn f() { let x = 5 }\nf()", "fn f(n) { if n < 2 { return n }\nf(n - 1) + f(n - 2) }\nf(15)",
		"let fact = fn(n) { if n == 0 { 1 } else { n * fact(n - 1) } }\nfact(10)",
		"fn make() { let c = 0\nfn() { c++\nc } }\nlet next = make()\nnext()\nnext()\nnext()",
		"fn adder(x) { fn(y) { x + y } }\nadder(2)(3)",
		"fn outer() { let a = 1\nlet inc = fn() { a = a + 10 }\ninc()\na }\nouter()",
		"fn f() { let f2 = fn(n) { if n == 0 { 0 } else { f2(n - 1) + 1 } }\nf2(5) }\nf()",
		"let fs = []\nrepeat i in 1..3 { fs.insert(fn() { i }) }\nfs.map(fn(f) { f() })",
		"let total = 0\nfn bump() { total++ }\nbump()\nbump()\ntotal",
		"fn f() { 1 }\nf", "len", "let x = 5\nx()", "fn f() { break }\nf()", "break", "repeat i in 3 { continue }",
		"fn f(n) { f(n + 1) }\nf(0)",
		"[1, 2, 3].map(fn(x) { x * 2 }).filter(fn(x) { x > 2 })", "[3, 1, 2].sort(fn(a, b) { b - a })",
		`let h = {"double": fn(x) { x * 2 }}` + "\nh.double(4)",
		"fn a() { b() }\nfn b() { 42 }\na()",
		"fn counter() { let n = 0\nreturn { \"inc\": fn() { n++ }, \"get\": fn() { n } } }\nlet c = counter()\nc.inc()\nc.inc()\nc.get()",
		"fn f(x) { fn() { x++ } }\nlet g = f(1)\ng()\ng()",
		"fn outer() { let v = 1\nfn mid() { fn inner() { v = v + 1 }\ninner() }\nmid()\nv }\nouter()",
	})
}

func TestLoops(t *testing.T) {
	expectGolden(t, []string{
		"let n = 0\nrepeat i in 1..10 { n = n + i }\nn", "let n = 0\nrepeat i in 5 { n = n + i }\nn",
		`let s = ""` + "\nrepeat c in \"héllo\" { s = c + s }\ns",
		`let ks = []` + "\nrepeat k in {\"b\": 1, \"a\": 2} { ks.insert(k) }\nks",
		"let n = 0\nrepeat n < 5 { n++ }\nn", "repeat i in 3 { i }",
		"let n = 0\nrepeat i in 1..100 { if i > 5 { break }\nn = i }\nn",
		"let n = 0\nrepeat i in 1..10 { if i % 2 == 0 { continue }\nn = n + i }\nn",
		"let out = []\nrepeat i in 1..3 { repeat j in 1..3 { if j > i { break }\nout.insert([i, j]) } }\nout",
		"fn f() { repeat i in 1..10 { if i == 4 { return i } } }\nf()",
		"let a = [1, 2]\nrepeat x in a { a.insert(x) }\na",
		"repeat x in true { }", "repeat i in 1..\"a\" { }",
		"let n = 0\nrepeat true { n++\nif n == 3 { break } }\nn",
		"let n = 0\nrepeat i in 1..5 { let sq = i * i\nn = n + sq }\nn",
		"let k = 0\nrepeat k < 3 { let inner = k\nk++ }\ninner",
		"let r = []\nrepeat i in 1..3 { print(i, [i]) }",
		"let total = 0\nrepeat i in 1..3 { total = total + [i, i].map(fn(x) { if x > 1 { return x } else { 0 } }).reduce(fn(a, b) { a + b }, 0) }\ntotal",
	})
}

func TestBuiltinsAndMethods(t *testing.T) {
	expectGolden(t, []string{
		`print("a", 1, [2, "b"], {"k": null})`, `printf("%d-%s", 1, "x")`, `len("abc")`,
		`type([])`, `toString(1.5)`, `"Hello".upper().lower()`, `"a,b".split(",")`,
		`[1, 2, 3].reduce(fn(a, b) { a + b }, 10)`, `[1, 2].join("-")`, `{"a": 1}.keys()`,
		`"abc".nope()`, `[1].nope`, `range(1, 3)`, `abs(-3)`, `max(1, 9)`,
		`let a = [1]` + "\na.insert(2)\na", `[1, 2, 3].find(fn(x) { x > 1 })`,
	})
}

func TestErrorsHavePositions(t *testing.T) {
	expectGolden(t, []string{
		"let a = 1\nlet b = a + true", "fn f() {\n  missing\n}\nf()", "let x = [1]\nx.foo()",
		"fn f(n) {\n  return n / 0\n}\nf(1)", "[1, 2].map(fn(x) {\n  x + \"a\" - 1\n})",
		"let h = {}\nh.a.b", `"a"["x"]`, "abs(\"x\")",
	})
}

// TestExamples checks the output of every example against
// testdata/examples, except math_string_time.tg, which prints the time.
func TestExamples(t *testing.T) {
	sourceDir, _ = filepath.Abs("../examples")
	defer func() { sourceDir = "" }()

	files, _ := os.ReadDir("../examples")
	for _, file := range files {
		name := file.Name()
		if !strings.HasSuffix(name, ".tg") || name == "math_string_time.tg" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile("../examples/" + name)
			if err != nil {
				t.Fatal(err)
			}
			got := runVM(t, string(source))
			text := got.output + "--- " + got.value + "\n"

			path := "testdata/examples/" + strings.TrimSuffix(name, ".tg") + ".golden"
			if *update {
				os.WriteFile(path, []byte(text), 0o644)
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%s, run go test ./vm -update", err)
			}
			if text != string(want) {
				t.Errorf("output differs from %s:\n%s", path, text)
			}
		})
	}
}
