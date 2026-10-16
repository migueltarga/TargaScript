package vm

import "testing"

// An argument that uses it is a function of it, unless it is a variable
func TestImplicitIt(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"[3, 1, 2].map(it * 2)", "[6, 2, 4]"},
		{"[3, 1, 2].filter(it > 1)", "[3, 2]"},
		{"[3, 1, 2].sortBy(-it)", "[3, 2, 1]"},
		{`[{name: "Ana", age: 30}, {name: "Bo", age: 20}].filter(it.age > 25).map(it.name)`, `["Ana"]`},
		// The innermost argument has its own it
		{"[[1, 2], [3]].map(it.map(it * 10))", "[[10, 20], [30]]"},
		{"[[1, 2], [3]].map(it.length + it.sum())", "[5, 4]"},
		// it alone as an argument is the it around it
		{"[1, 5].map(max(it, 2))", "[2, 5]"},
		// The right side of |> that uses it is a function of the value
		{"(1..6) |> it.map(n => n * n) |> it.filter(it % 2 == 0)", "[4, 16, 36]"},
		{`"  Hi There " |> it.trim().lower() |> it.split(" ")`, `["hi", "there"]`},
		{"[1, 2, 3] |> it.length + it.sum()", "9"},
		{"5 |> max(it, 3)", "5"},
		// Functions written out keep their parameters, even one named it
		{"[1, 2].map(x => x + 1)", "[2, 3]"},
		{"[1, 2].map((it) => it + 1)", "[2, 3]"},
		{"[1, 2].map(it => it + 1)", "[2, 3]"},
		{"[1, 2].map(fn(v) { v * 3 })", "[3, 6]"},
		{"fn twice(f, v) { f(f(v)) }\ntwice(it + 1, 5)", "7"},
		{`[1, 2].map(if it > 1 { "big" } else { "small" })`, `["small", "big"]`},
		{"[1, 2, 3].map(it * it).filter(it % 2 == 1)", "[1, 9]"},
		// A variable named it is a value
		{"let it = 10\n[1, 2].map(x => x + it)", "[11, 12]"},
		{"fn f(it) { [it, it + 1] }\nf(5)", "[5, 6]"},
		{"fn f() { let it = 3\n[1].map(fn(x) { x + it }) }\nf()", "[4]"},
		{"[1, 2].map(it * 2)\nlet it = 1\n[it]", "[1]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
	// A variable it is passed as it is
	if result := testEval("let it = 5\n[1].map(it)"); !isError(result) {
		t.Errorf("map(it) with a variable it should pass the number: %s", result.Inspect())
	}
}
