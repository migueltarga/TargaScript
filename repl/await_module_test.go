package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAwaitInLoadedFunctions(t *testing.T) {
	dir := t.TempDir()
	module := "fn slowDouble(n) {\n    await sleep(5)\n    n * 2\n}\nlet api = {get: fn(n) { await sleep(5); n + 1 }}\n"
	if err := os.WriteFile(filepath.Join(dir, "lib.tg"), []byte(module), 0o644); err != nil {
		t.Fatal(err)
	}
	SetMainDir(dir)
	defer SetMainDir("")

	var out bytes.Buffer
	program := `load "lib.tg" as lib
print(lib.slowDouble(21))
let f = lib.slowDouble
print(f(5))
print(lib.api.get(1))
load "lib.tg"
print(slowDouble(1))
`
	if err := EvalSource(program, &out, true, false, false); err != nil {
		t.Fatal(err)
	}
	if out.String() != "42\n10\n2\n2\n" {
		t.Errorf("printed %q", out.String())
	}
}
