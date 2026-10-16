package vm

import "testing"

func TestMorePatterns(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let name = \"Ana\"\nlet age = 30\n{name, age}", `{"name": "Ana", "age": 30}`},
		{"let x = 1\n{x, \"y\": 2}", `{"x": 1, "y": 2}`},
		{"let out = []\nrepeat [x, y] in [[1, 2], [3, 4]] { out.insert(x * y) }\nout", "[2, 12]"},
		{"let out = []\nrepeat {name, age} in [{\"name\": \"a\", \"age\": 1}] { out.insert(name + age) }\nout", `["a1"]`},
		{"let out = []\nrepeat i, [a, b] in [[1, 2], [3, 4]] { out.insert([i, a + b]) }\nout", "[[0, 3], [1, 7]]"},
		{"let out = []\nrepeat k, {v} in {\"x\": {\"v\": 1}} { out.insert([k, v]) }\nout", `[["x", 1]]`},
		{"fn area({width, height}) { width * height }\narea({\"width\": 2, \"height\": 3})", "6"},
		{"fn sum([a, b]) { a + b }\nsum([4, 5])", "9"},
		{"fn f(x, {y} = {\"y\": 7}) { x + y }\n[f(1), f(1, {\"y\": 2})]", "[8, 3]"},
		{"let pts = [[1, 2], [3, 4]]\npts.map(fn([x, y]) { x + y })", "[3, 7]"},
		{"fn f({a}) { fn() { a } }\nf({\"a\": 5})()", "5"},
		{"fn area({width, height}) { 1 }\narea", "fn area({width, height}) {\n{ 1 }\n}"},
		{"let fs = []\nrepeat [a] in [[1], [2]] { fs.insert(fn() { a }) }\nfs.map(fn(f) { f() })", "[1, 2]"},
		{"repeat [1, 2] in [] { }", "ERROR: parser errors: line 1:15: expected a name or a pattern like [a, b] before \"in\""},
		{"let n = 0\nrepeat [n].length > 0 && n < 3 { n++ }\nn", "3"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
