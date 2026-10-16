package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestTryCatch(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"try { 1 } catch e { 2 }", "1"},
		{"try { missing } catch e { e.message }", `"missing" is not defined`},
		{"try {\n  1 / 0\n} catch e { [e.line, e.column] }", "[2, 5]"},
		{`try { throw "bad input" } catch e { e.message }`, "bad input"},
		{`try { throw 42 } catch e { e.message }`, "42"},
		{"try { throw [1, 2] } catch { \"caught\" }", "caught"},
		{"fn f() { missing() }\ntry { f() } catch e { e.message }", `"missing" is not defined`},
		{"fn f(n) { if n == 0 { throw \"bottom\" }\nf(n - 1) }\ntry { f(50) } catch e { e.message }", "bottom"},
		{"let r = try { [1, 2].map(fn(x) { x.nope() }) } catch e { e.message }\nr", `cannot call method "nope" on an integer`},
		{"try { try { throw \"inner\" } catch e { throw e } } catch e { e.message + \"!\" }", "inner!"},
		{"try {\n  try { throw \"a\" } catch e { throw \"b\" }\n} catch e { e.message }", "b"},
		{"try {\n  throw \"x\"\n} catch e {\n  try { throw e } catch e2 { e2.line }\n}", "2"},
		{"let n = 0\nrepeat i in 1..5 { try { if i == 3 { throw \"skip\" }\nn = n + i } catch { } }\nn", "12"},
		{"let n = 0\nrepeat i in 1..5 { try { if i == 3 { break } } catch { }\nn = n + i }\ntry { throw \"after\" } catch e { [n, e.message] }", `[3, "after"]`},
		{"fn f() { repeat i in 1..3 { try { return i * 10 } catch { } } }\nf()", "10"},
		{"let x = 1\ntry { let x = 2\nthrow \"t\" } catch e { x }", "1"},
		{"try { 1 } catch e { 2 }\ne", `ERROR at line 2, column 1: "e" is not defined`},
		{"throw \"top\"", "ERROR at line 1, column 1: top"},
		{"try { throw \"a\" } catch e { throw \"from catch\" }", "ERROR at line 1, column 29: from catch"},
		{"let fs = []\nrepeat i in 1..3 { try { throw i } catch e { fs.insert(fn() { e.message }) } }\nfs.map(fn(f) { f() })", `["1", "2", "3"]`},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		got := result.Inspect()
		if s, ok := result.(*object.String); ok {
			got = s.Value
		}
		if got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}

func TestTryCatchAroundAwait(t *testing.T) {
	p := object.NewPromise()
	result, output := awaitRun(t, `
fn load() {
    try {
        let value = await p
        print("got", value)
    } catch e {
        print("failed:", e.message)
    }
    "done"
}
load()`, map[string]*object.Promise{"p": p})

	p.Reject(&object.String{Value: "offline"})
	if output.String() != "failed: offline\n" {
		t.Errorf("output: %q", output.String())
	}
	if result.Inspect() != "<promise: done>" {
		t.Errorf("run's promise: %s", result.Inspect())
	}
}
