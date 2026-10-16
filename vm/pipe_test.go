package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestPipeOperator(t *testing.T) {
	prelude := "fn double(x) { x * 2 }\nfn add(x, n) { x + n }\n"
	tests := []struct{ input, expected string }{
		{"5 |> double", "10"},
		{"5 |> add(3)", "8"},
		{"5 |> add(3) |> double", "16"},
		{"5\n    |> add(1)\n    |> double", "12"},
		{"[3, 1, 2] |> (l => l.sort()) |> toJSON", `"[1,2,3]"`},
		{`"x" |> fn(s) { s + "!" }`, `"x!"`},
		{"let m = {f: fn(a, b) { a - b }}; 10 |> m.f(4)", "6"},
		{"null ?? 2 |> double", "4"},
		{"let x = 3 |> double; x", "6"},
		{"[1, 2] |> add([3])", "[1, 2, 3]"},
		{"let args = [1]; 5 |> add(...args)", "6"},
	}
	for _, tt := range tests {
		got := testEval(prelude + tt.input)
		inspected := got.Inspect()
		if str, ok := got.(*object.String); ok {
			inspected = `"` + str.Value + `"`
		}
		if inspected != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, inspected, tt.expected)
		}
	}
}

func TestPipeErrors(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"5 |> 3", "cannot call an integer"},
		{"5 |>", "expected"},
	}
	for _, tt := range tests {
		got := testEval(tt.input)
		err, ok := got.(*object.Error)
		if !ok || !strings.Contains(err.Message, tt.expected) {
			t.Errorf("%s: got %v, want an error with %q", tt.input, got, tt.expected)
		}
	}
}
