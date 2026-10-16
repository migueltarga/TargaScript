package vm

import "testing"

// Methods see their object as this, whatever path their call takes
func TestMethodsSeeThis(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let o = {n: 1, get: fn() { this.n }}\no.get()", "1"},
		{"let o = {n: 1, add: fn(k) { this.n += k }}\no.add(2)\no.add(3)\no.n", "6"},
		{"let o = {n: 1, add: fn(k) { this.n += k ?? 10 }}\no.add()\no.n", "11"},
		{"let o = {n: 1, add: fn(k) { this.n += k }}\no.add(2, 3)\no.n", "3"},
		{"let o = {n: 1, all: fn(...ks) { this.n + ks.length }}\no.all(1, 2)", "3"},
		{"let f = fn() { this }\nf()", "null"},
		{"let o = {n: 2, f: fn() { this?.n }}\nlet g = o.f\ng() ?? \"none\"", "none"},
		{"let a = {n: 1, f: fn(o) { this.n + o.g() }}\nlet b = {n: 5, g: fn() { this.n }}\na.f(b)", "6"},
		{"let o = {n: 3, times: fn(xs) { xs.map(x => x * this.n) }}\no.times([1, 2])", "[3, 6]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}
