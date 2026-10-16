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
		{`print(try { exec("ls", "-l") } catch e { e.message })`, "exec takes a command, then a list of arguments or options like {\"dir\": \"src\"}, got a string\n"},
		{`print(exec("pwd", {"dir": "/"}).output)`, "/\n\n"},
		{`print(exec("tr a-z A-Z", {"input": "shout"}).output)`, "SHOUT\n"},
		{`print(exec("cat", [], {"input": "x"}).output)`, "x\n"},
		{`let r = exec("echo shown", {"show": true})
print(r.output == "", r.ok)`, "shown\ntrue true\n"},
		{`print(try { exec("ls", {"cwd": "/"}) } catch e { e.message })`, "exec has no option \"cwd\"; it has dir, input and show\n"},
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
