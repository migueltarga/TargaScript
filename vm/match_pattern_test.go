package vm

import "testing"

func TestMatchPatterns(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Array patterns bind their elements
		{"match [1, 2] { [x, y] => x + y }", "3"},
		{"match [0, 5] { [0, y] => \"on y at ${y}\", _ => \"no\" }", "on y at 5"},
		{"match [1, 2] { [_, y] => y }", "2"},
		{"match [1, 2, 3] { [x, y] => \"two\", [x, ...rest] => \"${x} then ${rest}\" }", "1 then [2, 3]"},
		{"match [1] { [x, ...rest] => rest }", "[]"},
		{"match [[1, 2], 3] { [[a, b], c] => a + b + c }", "6"},
		{"match [5, \"x\"] { [1..9, s] => s }", "x"},
		{"match [4, 1] { [2 * 2, y] => y }", "1"},
		{"match [true, 1] { [true, n] => n }", "1"},
		{"match [1, 2, 3] { [a, b] => \"two\", _ => \"other\" }", "other"},
		{"match [1] { [a, b, ...rest] => \"long\", _ => \"short\" }", "short"},
		{"match \"ab\" { [a, b] => \"array\", _ => \"other\" }", "other"},
		{"match null { [x] => 1, _ => 2 }", "2"},
		{"match [1, 2] { [1, 2] => \"pair\", _ => \"other\" }", "pair"},

		// Object patterns bind properties, and extra properties don't matter
		{"match {name: \"Ana\", age: 31} { {name, age} if age > 18 => \"${name} is an adult\", {name} => name }", "Ana is an adult"},
		{"match {name: \"Ana\", age: 3} { {name, age} if age > 18 => \"${name} is an adult\", {name} => name }", "Ana"},
		{"match {type: \"circle\", radius: 2} { {type: \"square\", side: s} => s * s, {type: \"circle\", radius: r} => 3 * r * r }", "12"},
		{"match {a: 1, b: 2, c: 3} { {a, ...rest} => rest }", `{"b": 2, "c": 3}`},
		{"match {\"first name\": \"Ana\"} { {\"first name\": n} => n }", "Ana"},
		{"match {a: 1} { {b} => \"b\", _ => \"none\" }", "none"},
		{"match {a: null} { {a} => \"present\", _ => \"none\" }", "present"},
		{"match 5 { {x} => 1, _ => 2 }", "2"},
		{"match {p: [1, 2]} { {p: [x, y]} => x * y }", "2"},

		// Bindings belong to the arm
		{"let x = 10\nlet r = match [1] { [x] => x }\n[r, x]", "[1, 10]"},
		{"let fs = match [1, 2] { [a, b] => [() => a, () => b] }\nfs.map(f => f())", "[1, 2]"},
		{"fn area(shape) { match shape { [w, h] => w * h, {r} => 3 * r * r, _ => 0 } }\n[area([2, 3]), area({r: 1}), area(7)]", "[6, 3, 0]"},
		{"let out = []\nrepeat p in [[1, 2], [0, 4]] { out.insert(match p { [0, y] => y, [x, y] => x * y }) }\nout", "[2, 4]"},
		{"match [1, 2] { [x], [x, _] => x }", "1"},
		{"fn f(a, b) { a + b }\nf(10, match [1, 2] { [x, y] => x * y })", "12"},
		{"match [1, [2, 3]] { [a, pair] => match pair { [b, c] => a + b + c } }", "6"},
		{"let p = [1, 2]\n[match p { [x, y] => x }, match p { [x, y] => y }]", "[1, 2]"},
		{"match [1] { [x] => x }\nx", `ERROR at line 2, column 1: "x" is not defined`},
		{"match [1] { [x], [y] => 1 }", `ERROR at line 1, column 18: the patterns of an arm must bind the same names, [x] binds x and [y] binds y`},
		{"match [1, 2] { [x, x] => x }", `ERROR at line 1, column 16: x is bound twice in [x, x]`},
		{"match [1] { [x] => x, [y, y] => 1 }", `ERROR at line 1, column 23: y is bound twice in [y, y]`},
		{"match [1] { 0, [x] => x }", `ERROR at line 1, column 16: the patterns of an arm must bind the same names, 0 binds nothing and [x] binds x`},
		{"match [1] { [x = 2] => x }", `ERROR at line 1, column 13: a match pattern can't give a default, [x = 2] takes the element as it is`},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
