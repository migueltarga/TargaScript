package vm

import "testing"

func TestObjectsAfterArrow(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let f = (x, y) => {x: x, y: y}; f(1, 2)", `{"x": 1, "y": 2}`},
		{`[1].map(n => {"n": n})`, `[{"n": 1}]`},
		{"let f = (x, y) => {x, y}; f(1, 2)", `{"x": 1, "y": 2}`},
		{"let f = o => {...o, b: 2}; f({a: 1})", `{"a": 1, "b": 2}`},
		{"match 1 { 1 => {one: true}, _ => null }", `{"one": true}`},
		// Blocks stay blocks
		{"let f = n => { n * 2 }; f(3)", "6"},
		{"let f = n => { n }; f(3)", "3"},
		{"let f = n => {}; f(3)", "null"},
		{"let f = n => { let a = n; a + 1 }; f(3)", "4"},
		{"match 1 { 1 => { let a = 5; a }, _ => 0 }", "5"},
		{"let f = n => ({a: n}); f(1)", `{"a": 1}`},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}
