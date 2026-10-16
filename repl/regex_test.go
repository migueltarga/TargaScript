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

func TestReplacePatternWithAFunction(t *testing.T) {
	program := `let data = {name: "Ana", count: 3}
print("Hi {{name}}, {{count}} new".replacePattern("\{\{(\w+)\}\}", (whole, key) => data[key]))
print("a1b22".replacePattern("(\d)(\d)?", (all, a, b) => "[${a}|${b}]"))
print("prices: 10, 25".replacePattern("\d+", n => toInt(n) * 2))
print("none here".replacePattern("\d", d => "x"))
print(try { "ab".replacePattern("a", () => null.y) } catch e { e.message })
print(try { "ab".replacePattern("a", 5) } catch e { e.message })
`
	expected := `Hi Ana, 3 new
a[1|]b[2|2]
prices: 20, 50
none here
cannot read property "y" of null
replacement for string.replacePattern() must be a string or a function, got an integer
`
	var out bytes.Buffer
	if err := EvalSource(program, &out, true, false, false); err != nil {
		t.Fatal(err)
	}
	if out.String() != expected {
		t.Errorf("printed:\n%s\nwant:\n%s", out.String(), expected)
	}
}
