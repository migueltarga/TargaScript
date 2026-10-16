package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

const pointStruct = `struct Point {
  x = 0, y = 0
  fn len() { this.x * this.x + this.y * this.y }
  fn move(dx, dy) { Point(this.x + dx, this.y + dy) }
}
`

// A struct makes values with its fields in order and gives them its methods
func TestStruct(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"Point(3, 4)", `Point {"x": 3, "y": 4}`},
		{"Point()", `Point {"x": 0, "y": 0}`},
		{"Point(1)", `Point {"x": 1, "y": 0}`},
		{"Point(3, 4).len()", "25"},
		{"Point(3, 4).move(1, 1)", `Point {"x": 4, "y": 5}`},
		{"let p = Point(3, 4)\np.x = 10\np.len()", "116"},
		{"[Point(1, 2), Point(2, 3)].map(it.len())", "[5, 13]"},
		{"[Point(1, 2) == Point(1, 2), Point(1, 2) == {x: 1, y: 2}]", "[true, false]"},
		{"[type(Point), type(Point())]", `["struct", "Point"]`},
		{"Point(1).z", "null"},
		{"let p = {x: 1, len: fn() { 7 }}\np.len()", "7"},
	}
	for _, tt := range tests {
		if got := testEval(pointStruct + tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}

// A struct declared in a function sees its own name and the function's locals
func TestLocalStruct(t *testing.T) {
	input := `fn make(base) {
  struct Counter {
    n = base
    fn inc() { Counter(this.n + 1) }
  }
  Counter
}
make(10)().inc().inc()`
	if got := testEval(input).Inspect(); got != `Counter {"n": 12}` {
		t.Errorf("got %s", got)
	}
}

func TestStructErrors(t *testing.T) {
	tests := []struct{ input, message string }{
		{pointStruct + "Point(1, 2, 3)", "wrong number of arguments for Point: got 3, want up to 2"},
		{pointStruct + "struct Point {}", "identifier already declared: Point"},
		{pointStruct + "Point = 1", "cannot reassign constant: Point"},
	}
	for _, tt := range tests {
		result, ok := testEval(tt.input).(*object.Error)
		if !ok || !strings.Contains(result.Message, tt.message) {
			t.Errorf("%s: got %v, want %q", tt.input, result, tt.message)
		}
	}
}
