package repl

import (
	"bytes"
	"testing"

	"github.com/migueltarga/TargaScript/evaluator"
)

func TestTestsCanAwait(t *testing.T) {
	evaluator.ResetTests()
	defer evaluator.ResetTests()
	var out bytes.Buffer
	program := `test("slow pass", fn() { await sleep(10); assertEqual(1, 1) })
test("slow fail", fn() {
    await sleep(1)
    assertEqual(1, 2)
})
test("sync", fn() { assert(true) })
print("after")
`
	if err := EvalSource(program, &out, true, false, false); err != nil {
		t.Fatal(err)
	}
	want := "✓ slow pass\n✗ slow fail\n    assertion failed: expected 2, got 1 (line 4)\n✓ sync\nafter\n"
	if out.String() != want {
		t.Errorf("printed:\n%s\nwant:\n%s", out.String(), want)
	}
	if passed, failed := evaluator.TestResults(); passed != 2 || failed != 1 {
		t.Errorf("results: %d passed, %d failed", passed, failed)
	}
}
