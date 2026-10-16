package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestThis(t *testing.T) {
	counter := "let counter = {count: 0, inc: fn() { this.count++; this }, total: fn(by = 1) { this.count * by }}\n"
	tests := []struct{ input, expected string }{
		{counter + "counter.inc().inc().count", "2"},
		{counter + "counter.inc(); counter.total(10)", "10"},
		{counter + "counter.inc(); counter.total(...[3])", "3"},
		// Arrow functions see the this of the function around them
		{"let o = {n: 2, f: fn() { [1, 2].map(x => x * this.n) }}; o.f()", "[2, 4]"},
		// A nested fn has its own
		{"let o = {n: 2, f: fn() { let g = fn() { this }; g() }}; o.f()", "null"},
		{"fn make(n) { {n, get: fn() { this.n }} }; make(5).get()", "5"},
		{"this", "null"},
		{"fn plain() { this }; plain()", "null"},
		{"let o = {f: fn() { this }}; let g = o.f; g()", "null"},
		{"let o = {f: fn() { this == o }}; o.f()", "true"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}

	err, ok := testEval("let o = {f: fn() { this = 1 }}; o.f()").(*object.Error)
	if !ok || !strings.Contains(err.Message, "this") {
		t.Errorf("assigning this: got %v", err)
	}
}
