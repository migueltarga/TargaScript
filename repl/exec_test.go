package repl

import (
	"bytes"
	"runtime"
	"testing"
)

func TestExec(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	tests := []struct{ code, want string }{
		{`let r = exec("echo hi; echo oops >&2; exit 3")
print(r.output.trim(), r.errors.trim(), r.code, r.ok)`, "hi oops 3 false\n"},
		{`print(exec("echo", ["a b", 1]).output)`, "a b 1\n\n"},
		{`print(exec("true").ok)`, "true\n"},
		{`print(try { exec("no-such-program-here", []) } catch e { e.message })`, "cannot run \"no-such-program-here\": no such program\n"},
		{`print(try { exec("ls", "-l") } catch e { e.message })`, "second argument to `exec` must be an array of arguments, got a string\n"},
	}
	for _, tt := range tests {
		var out bytes.Buffer
		if err := EvalSource(tt.code, &out, true, false, false); err != nil {
			t.Errorf("%s: %v", tt.code, err)
			continue
		}
		if out.String() != tt.want {
			t.Errorf("%s: got %q, want %q", tt.code, out.String(), tt.want)
		}
	}
}
