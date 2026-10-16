package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestAwaitAll(t *testing.T) {
	a, b := object.NewPromise(), object.NewPromise()
	result, out := awaitRun(t, "let r = await all([a, 2, b])\nprint(r)", map[string]*object.Promise{"a": a, "b": b})
	promise, ok := result.(*object.Promise)
	if !ok {
		t.Fatalf("expected the program to pause, got %s", result.Inspect())
	}
	// Settled out of order, the results keep the order of the array
	b.Resolve(&object.String{Value: "b"})
	if promise.Settled() {
		t.Fatal("resumed before every promise settled")
	}
	a.Resolve(&object.Integer{Value: 1})
	if out.String() != "[1, 2, \"b\"]\n" {
		t.Errorf("printed %q", out.String())
	}
}

func TestAwaitAllSettledAlready(t *testing.T) {
	a, b := object.NewPromise(), object.NewPromise()
	a.Resolve(&object.Integer{Value: 1})
	result, out := awaitRun(t, "print(await all([a, b]))", map[string]*object.Promise{"a": a, "b": b})
	if _, ok := result.(*object.Promise); !ok {
		t.Fatalf("resolved before b settled: %s, printed %q", result.Inspect(), out.String())
	}
	b.Resolve(&object.Integer{Value: 2})
	if out.String() != "[1, 2]\n" {
		t.Errorf("printed %q", out.String())
	}
}

func TestAwaitAllRejected(t *testing.T) {
	a, b := object.NewPromise(), object.NewPromise()
	result, out := awaitRun(t, "try { await all([a, b]) } catch e { print(e.message) }", map[string]*object.Promise{"a": a, "b": b})
	if _, ok := result.(*object.Promise); !ok {
		t.Fatalf("expected the program to pause, got %s", result.Inspect())
	}
	b.Reject(&object.Error{Message: "b failed"})
	if out.String() != "b failed\n" {
		t.Errorf("printed %q", out.String())
	}
}

func TestAllWithoutPromises(t *testing.T) {
	result, out := awaitRun(t, "print(await all([1, 2]), await all([]))", nil)
	if result.Type() == object.ERROR_OBJ {
		t.Fatal(result.Inspect())
	}
	if out.String() != "[1, 2] []\n" {
		t.Errorf("printed %q", out.String())
	}
	if err, ok := testEval("all(1)").(*object.Error); !ok || err.Message != "argument to all must be an array of promises, got an integer" {
		t.Errorf("all(1): %v", testEval("all(1)"))
	}
}
