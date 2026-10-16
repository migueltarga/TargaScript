package vm

import "testing"

// |> into a name that isn't a function calls the method of the value
func TestPipeIntoMethods(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"(1..10) |> map(it * it) |> filter(it % 2 == 0)", "[4, 16, 36, 64, 100]"},
		{`"  Hi There " |> trim |> lower |> split(" ")`, `["hi", "there"]`},
		{`[3, 1, 2] |> sortBy(-it) |> join("-")`, "3-2-1"},
		{"{a: 1, b: 2} |> keys", `["a", "b"]`},
		{"[1, 2] |> map(...[x => x + 1])", "[2, 3]"},
		// Functions the program has still get the value first
		{"fn double(xs) { xs.map(it * 2) }\n[1, 2] |> double |> reverse", "[4, 2]"},
		{"[4, 9] |> max", "9"},
		{"fn map(xs, f) { \"mine\" }\n[1] |> map(it)", "mine"},
		{"let add = (a, b) => a + b\n1 |> add(2)", "3"},
		{"[1, 2] |> (xs => xs.length)", "2"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
	if !isError(testEval("[1] |> mpa(it)")) {
		t.Error("a method arrays don't have should fail")
	}
}
