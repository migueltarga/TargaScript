package object

import (
	"strings"
	"testing"
)

func TestConcatSharesSafely(t *testing.T) {
	base := Concat(&String{Value: strings.Repeat("x", 70)}, "!")
	if base.buffer == nil {
		t.Fatal("a long string should get a buffer")
	}
	// Two strings made from the same one: the first writes after it in the
	// buffer, the second must not overwrite that
	a := Concat(base, "A")
	b := Concat(base, "B")
	c := Concat(a, "C")
	want := strings.Repeat("x", 70) + "!"
	if base.Value != want || a.Value != want+"A" || b.Value != want+"B" || c.Value != want+"AC" {
		t.Errorf("base %q\na %q\nb %q\nc %q", base.Value[70:], a.Value[70:], b.Value[70:], c.Value[70:])
	}

	// Building a string in a loop keeps every step as it was
	steps := []*String{{Value: ""}}
	for i := range 500 {
		steps = append(steps, Concat(steps[len(steps)-1], string(rune('a'+i%26))))
	}
	for i, step := range steps {
		if len(step.Value) != i {
			t.Fatalf("step %d has length %d", i, len(step.Value))
		}
		if i > 0 && step.Value[:i-1] != steps[i-1].Value {
			t.Fatalf("step %d changed the one before", i)
		}
	}

	// Hashes of strings sharing a buffer are their own
	if a.HashKey() == b.HashKey() {
		t.Error("different strings, same hash")
	}
	if Concat(a, "") != a {
		t.Error("adding nothing should give the same string")
	}
}
