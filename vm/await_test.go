package vm

import (
	"bytes"
	"os"
	"testing"

	"github.com/migueltarga/TargaScript/compiler"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
)

// awaitRun starts a program with the given promises as globals and returns
// its result, its output so far and the output buffer.
func awaitRun(t *testing.T, input string, promises map[string]*object.Promise) (object.Object, *bytes.Buffer) {
	t.Helper()
	var out bytes.Buffer
	evaluator.SetOutput(&out)
	t.Cleanup(func() { evaluator.SetOutput(os.Stdout) })

	program := parser.New(lexer.New(input)).ParseProgram()
	c := compiler.New()
	machine := New(c.Globals())
	for name, p := range promises {
		machine.SetGlobal(name, p)
	}
	bytecode, err := c.Compile(program)
	if err != nil {
		t.Fatal(err)
	}
	return machine.Run(bytecode), &out
}

func TestAwaitPausesUntilResolved(t *testing.T) {
	p := object.NewPromise()
	result, out := awaitRun(t, `
print("before")
let value = await p
print("after", value)
value * 2`, map[string]*object.Promise{"p": p})

	promise, ok := result.(*object.Promise)
	if !ok {
		t.Fatalf("expected the run to return a promise, got %s", result.Inspect())
	}
	if out.String() != "before\n" {
		t.Fatalf("ran past the await: %q", out.String())
	}

	p.Resolve(&object.Integer{Value: 21})
	if out.String() != "before\nafter 21\n" {
		t.Fatalf("did not resume: %q", out.String())
	}
	if promise.Inspect() != "<promise: 42>" {
		t.Errorf("run's promise: %s", promise.Inspect())
	}
}

func TestAwaitKeepsLocalsLoopsAndCalls(t *testing.T) {
	a, b := object.NewPromise(), object.NewPromise()
	result, out := awaitRun(t, `
fn fetchBoth() {
    let results = []
    repeat p in [a, b] {
        let value = await p
        results.insert(value * 10)
    }
    return results
}
fn main() {
    let start = "x"
    let got = fetchBoth()
    print(start, got)
    got.length
}
main()`, map[string]*object.Promise{"a": a, "b": b})

	a.Resolve(&object.Integer{Value: 1})
	if out.Len() != 0 {
		t.Fatalf("finished before b resolved: %q", out.String())
	}
	b.Resolve(&object.Integer{Value: 2})
	if out.String() != "x [10, 20]\n" {
		t.Fatalf("got %q", out.String())
	}
	if result.Inspect() != "<promise: 2>" {
		t.Errorf("run's promise: %s", result.Inspect())
	}
}

func TestAwaitRejected(t *testing.T) {
	p := object.NewPromise()
	result, out := awaitRun(t, "print(1)\nlet v = await p\nprint(2)", map[string]*object.Promise{"p": p})

	p.Reject(&object.String{Value: "network down"})
	if out.String() != "1\n" {
		t.Fatalf("continued after the rejection: %q", out.String())
	}
	want := "<promise rejected: ERROR at line 2, column 9: network down>"
	if result.Inspect() != want {
		t.Errorf("got %s, want %s", result.Inspect(), want)
	}
}

func TestAwaitAlreadySettledAndPlainValues(t *testing.T) {
	p := object.NewPromise()
	p.Resolve(&object.String{Value: "ready"})

	result, out := awaitRun(t, `print(await p, await 5, await "x" + "y")`, map[string]*object.Promise{"p": p})
	if out.String() != "ready 5 xy\n" {
		t.Errorf("got %q", out.String())
	}
	if result.Inspect() != "<promise: null>" {
		t.Errorf("got %s", result.Inspect())
	}
}

func TestAwaitInBuiltinCallback(t *testing.T) {
	p := object.NewPromise()
	result, _ := awaitRun(t, "[1].filter(fn(x) {\n  await p\n})", map[string]*object.Promise{"p": p})

	err, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("expected an error, got %s", result.Inspect())
	}
	if err.Message != "await can't pause here: this function was called by a builtin such as filter or sort" || err.Line != 2 {
		t.Errorf("got %s", err.Inspect())
	}
}

func TestClosureCalledFromGoReturnsPromise(t *testing.T) {
	p := object.NewPromise()
	result, out := awaitRun(t, "fn(x) { let v = await p\nprint(x + v)\nx + v }", map[string]*object.Promise{"p": p})
	closure := result.(*object.Closure)

	// As JavaScript calls an event handler, once the program has finished
	called := evaluator.CallFunction(closure, []object.Object{&object.Integer{Value: 1}})
	promise, ok := called.(*object.Promise)
	if !ok {
		t.Fatalf("expected a promise, got %s", called.Inspect())
	}
	p.Resolve(&object.Integer{Value: 2})
	if out.String() != "3\n" || promise.Inspect() != "<promise: 3>" {
		t.Errorf("got output %q and %s", out.String(), promise.Inspect())
	}
}

func TestAwaitParses(t *testing.T) {
	expectAwait := map[string]string{
		"await f(1).g()": "await f(1).g()",
		"await p + 1":    "(await p + 1)",
	}
	for input, want := range expectAwait {
		program := parser.New(lexer.New(input)).ParseProgram()
		if got := program.String(); got != want {
			t.Errorf("%q parsed as %q, want %q", input, got, want)
		}
	}
}
