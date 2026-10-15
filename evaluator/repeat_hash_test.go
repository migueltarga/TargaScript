package evaluator

import "testing"

func TestRepeatOverHash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`let h = {"b": 2, "a": 1, "c": 3}
let keys = []
repeat k in h { keys = keys.insert(k) }
keys`, "[b, a, c]"},
		{`let h = {"x": 10, "y": 20}
let total = 0
repeat k in h { total = total + h[k] }
total`, "30"},
		{`let h = {1: "one", true: "yes"}
let out = []
repeat k in h { out = out.insert(type(k)) }
out`, "[INTEGER, BOOLEAN]"},
		{`let h = {"a": 1}
repeat k in h { h[k + "2"] = 2 }
h`, "{a: 1, a2: 2}"},
		{`let n = 0
repeat k in {} { n = n + 1 }
n`, "0"},
		{`let found = ""
repeat k in {"a": 1, "b": 2, "c": 3} {
  if k == "b" {
    found = k
    break
  }
}
found`, "b"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}
}
