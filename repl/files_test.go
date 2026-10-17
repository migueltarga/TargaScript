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

// Files that aren't text are read and written as bytes, unchanged
func TestFileBytes(t *testing.T) {
	t.Chdir(t.TempDir())
	code := `let data = bytes([0, 255, 0xc3, 10, 13])
writeFile("data.bin", data)
appendFile("data.bin", bytes([7]))
let back = readBytes("data.bin")
print(back, back == data + bytes([7]), type(readFile("data.bin")))`
	want := "bytes(00 ff c3 0a 0d 07) true string\n"

	var out bytes.Buffer
	if err := EvalSource(code, &out, true, false, false); err != nil {
		t.Fatal(err)
	}
	if out.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}
