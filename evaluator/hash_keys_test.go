package evaluator

import "testing"

func TestHashLiteralKeysOfDifferentTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`let h = {1: "int", "1": "str"}
[h[1], h["1"], h.length]`, `["int", "str", 2]`},
		{`{true: "bool", "true": "str"}.length`, "2"},
		{`{a: 1, "a": 2}`, `{"a": 2}`},
		{`{1: "a", 1: "b", 2: "c"}`, `{1: "b", 2: "c"}`},
		{`let n = 0
fn next() { n = n + 1
return n }
{next(): "x", next(): "y"}`, `{1: "x", 2: "y"}`},
		{`{1 + 1: "a", 2: "b"}`, `{2: "b"}`},
		{`{"a": 1, "b": 2,}`, `{"a": 1, "b": 2}`},
		{"{\n  \"a\": 1,\n  \"b\": 2,\n}", `{"a": 1, "b": 2}`},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}
}
