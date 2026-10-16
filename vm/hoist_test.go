package vm

import "testing"

func TestFunctionsInFunctionsCallEachOther(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"fn outer() {\n  fn a() { b() }\n  fn b() { 2 }\n  a()\n}\nouter()", "2"},
		{"fn outer() {\n  fn even(n) { if n == 0 { true } else { odd(n - 1) } }\n  fn odd(n) { if n == 0 { false } else { even(n - 1) } }\n  [even(10), odd(7)]\n}\nouter()", "[true, true]"},
		{"fn outer() {\n  if true {\n    fn a() { b() }\n    fn b() { 3 }\n    a()\n  }\n}\nouter()", "3"},
		// A function declared again in the same block still replaces it
		{"fn outer() {\n  fn a() { 1 }\n  let first = a()\n  fn a() { 2 }\n  [first, a()]\n}\nouter()", "[1, 2]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}
