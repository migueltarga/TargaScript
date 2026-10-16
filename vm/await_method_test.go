package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestAwaitInObjectMethods(t *testing.T) {
	p := object.NewPromise()
	input := `
let api = {
    base: 10,
    fetchItem: fn(n) {
        let v = await p
        v + n
    },
}
print(api.fetchItem(1))
let args = [2]
print(api.fetchItem(...args))
`
	result, out := awaitRun(t, input, map[string]*object.Promise{"p": p})
	if _, ok := result.(*object.Promise); !ok {
		t.Fatalf("expected the program to pause, got %s", result.Inspect())
	}
	p.Resolve(&object.Integer{Value: 40})
	if out.String() != "41\n42\n" {
		t.Errorf("printed %q", out.String())
	}
}

func TestObjectMethodsStillWork(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let o = {n: 2, double: fn(x) { x * 2 }}; o.double(4)", "8"},
		{"let o = {keys: fn() { 1 }}; o.keys()", "1"},
		{"let o = {get: fn(k) { \"own\" }}; o.get(\"a\")", `"own"`},
		{"let o = {n: 1}; o.missing()", `ERROR at line 1, column 18: object has no method "missing"`},
		{"let o = {n: 1}; o.n()", `ERROR at line 1, column 18: property "n" is an integer, not a function`},
		{"let o = {f: fn(a, b) { a - b }}; o.f(...[5, 3])", "2"},
	}
	for _, tt := range tests {
		got := testEval(tt.input)
		inspected := got.Inspect()
		if got.Type() == "STRING" {
			inspected = `"` + inspected + `"`
		}
		if inspected != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, inspected, tt.expected)
		}
	}
}
