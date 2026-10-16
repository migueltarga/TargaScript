package vm

import "testing"

func TestDefaultParameters(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`fn greet(name, greeting = "Hello") { "${greeting}, ${name}" }` + "\ngreet(\"Ana\")", "Hello, Ana"},
		{`fn greet(name, greeting = "Hello") { "${greeting}, ${name}" }` + "\ngreet(\"Ana\", \"Hi\")", "Hi, Ana"},
		{`fn greet(name, greeting = "Hello") { "${greeting}, ${name}" }` + "\ngreet(\"Ana\", null)", "Hello, Ana"},
		{"fn f(a, b = a * 2, c = a + b) { [a, b, c] }\nf(1)", "[1, 2, 3]"},
		{"fn f(a = 1, b = 2) { a + b }\nf()", "3"},
		{"fn f(a = 1, b = 2) { a + b }\nf(10)", "12"},
		{"let f = fn(x, step = 1) { x + step }\n[f(1), f(1, 5)]", "[2, 6]"},
		{"fn f(list = []) { list.insert(1)\nlist.length }\n[f(), f(), f([0])]", "[1, 1, 2]"},
		{"fn make(start = 0) { let n = start\nfn() { n++ } }\nlet next = make(10)\nnext()\nnext()", "12"},
		{"fn f(n, acc = 1) { if n <= 1 { return acc }\nf(n - 1, acc * n) }\nf(5)", "120"},
		{"fn f(a = missing) { a }\nf(1)", "1"},
		{"fn f(a = missing) { a }\nf()", `ERROR at line 1, column 10: "missing" is not defined`},
		{"fn f(a, b = 2) { a + b }\nf", "fn f(a, b = 2)"},
		{"[1, 2].map(fn(x, i = 100) { x + i })", "[1, 3]"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
