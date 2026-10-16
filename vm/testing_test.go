package vm

import (
	"bytes"
	"os"
	"testing"

	"github.com/migueltarga/TargaScript/evaluator"
)

func TestTestBuiltin(t *testing.T) {
	var out bytes.Buffer
	evaluator.SetOutput(&out)
	defer evaluator.SetOutput(os.Stdout)
	evaluator.ResetTests()
	defer evaluator.ResetTests()

	testEval(`
test("passes", fn() { assertEqual(1 + 1, 2) })
test("fails", fn() {
    assertEqual(1 + 1, 3)
})
test("throws", () => { throw "boom" })
print("still running")
`)

	want := "✓ passes\n" +
		"✗ fails\n    assertion failed: expected 3, got 2 (line 4)\n" +
		"✗ throws\n    boom (line 6)\n" +
		"still running\n"
	if out.String() != want {
		t.Errorf("output:\n%s\nwant:\n%s", out.String(), want)
	}
	if passed, failed := evaluator.TestResults(); passed != 1 || failed != 2 {
		t.Errorf("results: %d passed, %d failed, want 1 and 2", passed, failed)
	}
}

func TestTestBuiltinErrors(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`test("x")`, "wrong number of arguments for test. got=1, want=2"},
		{`test(1, fn() {})`, "first argument to `test` must be a string, got an integer"},
		{`test("x", 1)`, "second argument to `test` must be a function, got an integer"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); !bytes.Contains([]byte(got), []byte(tt.expected)) {
			t.Errorf("%s: got %s, want %q", tt.input, got, tt.expected)
		}
	}
}
