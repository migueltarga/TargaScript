package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestAwaitInMapGivesPromises(t *testing.T) {
	a, b := object.NewPromise(), object.NewPromise()
	input := `
fn tenTimes(p) {
    let v = await p
    v * 10
}
let started = [a, b].map(tenTimes)
print(started.length, type(started[0]))
print(await all(started))
`
	result, out := awaitRun(t, input, map[string]*object.Promise{"a": a, "b": b})
	if _, ok := result.(*object.Promise); !ok {
		t.Fatalf("expected the program to pause, got %s", result.Inspect())
	}
	if out.String() != "2 promise\n" {
		t.Fatalf("printed %q before the promises settled", out.String())
	}
	b.Resolve(&object.Integer{Value: 2})
	a.Resolve(&object.Integer{Value: 1})
	if out.String() != "2 promise\n[10, 20]\n" {
		t.Errorf("printed %q", out.String())
	}
}

func TestAwaitInMapCanPauseTwice(t *testing.T) {
	a, b := object.NewPromise(), object.NewPromise()
	input := `
let results = [1].map(fn(x) {
    let first = await a
    let second = await b
    x + first + second
})
print(await all(results))
`
	_, out := awaitRun(t, input, map[string]*object.Promise{"a": a, "b": b})
	a.Resolve(&object.Integer{Value: 10})
	b.Resolve(&object.Integer{Value: 100})
	if out.String() != "[111]\n" {
		t.Errorf("printed %q", out.String())
	}
}

func TestMapWithoutAwaitIsUnchanged(t *testing.T) {
	if got := testEval("[1, 2].map(x => x * 2)").Inspect(); got != "[2, 4]" {
		t.Errorf("got %s", got)
	}
}

func TestAwaitInMapRejected(t *testing.T) {
	a := object.NewPromise()
	input := `
let results = [1].map(fn(x) { await a })
try { await all(results) } catch e { print("caught", e.message) }
`
	_, out := awaitRun(t, input, map[string]*object.Promise{"a": a})
	a.Reject(&object.Error{Message: "nope"})
	if out.String() != "caught nope\n" {
		t.Errorf("printed %q", out.String())
	}
}
