package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestCollectionMethods(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`{"a": 1, "b": 2}.entries()`, `[["a", 1], ["b", 2]]`},
		{`{"a": 1}.get("a")`, "1"},
		{`{"a": 1}.get("z")`, "null"},
		{`{"a": 1}.get("z", 0)`, "0"},
		{`let h = {"a": 1, "b": 2, "c": 3}` + "\nlet removed = h.remove(\"b\")\n[removed, h, h.keys()]", `[2, {"a": 1, "c": 3}, ["a", "c"]]`},
		{`let h = {"a": 1}` + "\n[h.remove(\"z\"), h]", `[null, {"a": 1}]`},
		{"fn counter() { let n = 0\n{\"get\": fn() { n }, \"remove\": fn() { \"own\" }} }\nlet c = counter()\n[c.get(), c.remove()]", `[0, "own"]`},
		{"let a = [1, 2, 3]\n[a.pop(), a]", "[3, [1, 2]]"},
		{"let a = [1, 2, 3]\n[a.shift(), a]", "[1, [2, 3]]"},
		{"let a = []\n[a.pop(), a.shift()]", "[null, null]"},
		{"let a = [2, 3]\na.unshift(1)", "[1, 2, 3]"},
		{"let a = [1, 2, 3]\n[a.removeAt(1), a]", "[2, [1, 3]]"},
		{"let a = [1, 2, 3]\n[a.removeAt(-1), a]", "[3, [1, 2]]"},
		{"[1].removeAt(5)", "ERROR: position 5 is outside the array, which has 1 elements"},
		{`format("%05.1f|%-4s|%d", 3.14159, "ab", 42)`, "003.1|ab  |42"},
		{`format("no args")`, "no args"},
		{`format("%d")`, "ERROR: not enough arguments for printf: format has more verbs than the 0 given"},
		{`{"a": 1}.entires()`, `ERROR: object has no method "entires". Did you mean "entries"?`},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		got := result.Inspect()
		if s, ok := result.(*object.String); ok {
			got = s.Value
		}
		if err, ok := result.(*object.Error); ok {
			got = "ERROR: " + err.Message
		}
		if got != tt.expected {
			t.Errorf("%s\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
