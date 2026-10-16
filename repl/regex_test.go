package repl

import (
	"bytes"
	"testing"
)

func TestPatterns(t *testing.T) {
	program := `let t = "Ana 31, Bia 25, Caio 40"
print(t.matches("\d+"), "abc".matches("\d"))
print(t.match("(\w+) (\d+)"), "x".match("y"))
print(t.matchAll("(\w+) (\d+)").map(([_, name, age]) => name + age))
print(t.replacePattern("(\w+) (\d+)", "$2:$1"))
print("a1b22c".splitPattern("\d+"))
print(try { "x".matches("(") } catch e { e.message.startsWith("invalid pattern \"(\"") })
print(try { "x".matches(1) } catch e { e.message })
`
	expected := `true false
["Ana 31", "Ana", "31"] null
["Ana31", "Bia25", "Caio40"]
31:Ana, 25:Bia, 40:Caio
["a", "b", "c"]
true
pattern for string.matches() must be a string, got an integer
`
	var out bytes.Buffer
	if err := EvalSource(program, &out, true, false, false); err != nil {
		t.Fatal(err)
	}
	if out.String() != expected {
		t.Errorf("got:\n%s\nwant:\n%s", out.String(), expected)
	}
}
