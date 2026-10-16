package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesAndEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	t.Setenv("TARGASCRIPT_TEST", "yes")

	program := strings.ReplaceAll(`
writeFile("PATH", "first\n")
appendFile("PATH", "second\n")
print(readFile("PATH").lines())
print(listDir("DIR"))
print(env("TARGASCRIPT_TEST"), env("TARGASCRIPT_MISSING"), env("TARGASCRIPT_MISSING", "fallback"))
print(try { readFile("DIR/missing.txt") } catch e { e.message.startsWith("cannot read") })
`, "PATH", path)
	program = strings.ReplaceAll(program, "DIR", dir)

	var out bytes.Buffer
	if err := EvalSource(program, &out, true, false, false); err != nil {
		t.Fatal(err)
	}
	expected := `["first", "second"]
["notes.txt"]
yes null fallback
true
`
	if out.String() != expected {
		t.Errorf("got:\n%s\nwant:\n%s", out.String(), expected)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "first\nsecond\n" {
		t.Errorf("file content: %q", data)
	}
}
