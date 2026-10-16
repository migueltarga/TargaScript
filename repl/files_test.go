package repl

import (
	"bytes"
	"testing"
)

func TestFileHelpers(t *testing.T) {
	t.Chdir(t.TempDir())
	code := `print(exists("out"))
makeDir("out/a/b")
print(exists("out/a/b"))
writeFile("out/a/b/x.txt", "x")
print(try { deleteFile("out/a/b") } catch e { e.message })
deleteFile("out/a/b/x.txt")
deleteFile("out/a/b")
print(exists("out/a/b"), exists("out/a"))
print(try { deleteFile("missing.txt") } catch e { e.message })`
	want := "false\ntrue\ncannot delete \"out/a/b\": directory not empty\nfalse true\ncannot delete \"missing.txt\": no such file or directory\n"

	var out bytes.Buffer
	if err := EvalSource(code, &out, true, false, false); err != nil {
		t.Fatal(err)
	}
	if out.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}
