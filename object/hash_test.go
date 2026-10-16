package object

import (
	"fmt"
	"testing"
)

func TestHashKeepsOrderAcrossItsIndex(t *testing.T) {
	for _, size := range []int{3, indexAbove, indexAbove + 1, 40} {
		h := NewHash()
		for i := range size {
			h.Set(&String{Value: fmt.Sprint("k", i)}, &Integer{Value: int64(i)})
		}
		// Updating keeps the position
		h.Set(&String{Value: "k0"}, &Integer{Value: 100})
		if h.Len() != size {
			t.Fatalf("size %d: Len() = %d", size, h.Len())
		}
		for i := range size {
			pair, ok := h.Get((&String{Value: fmt.Sprint("k", i)}).HashKey())
			want := int64(i)
			if i == 0 {
				want = 100
			}
			if !ok || pair.Value.(*Integer).Value != want {
				t.Fatalf("size %d: k%d = %v, %t", size, i, pair.Value, ok)
			}
			if key, at := h.At(i); key != (&String{Value: fmt.Sprint("k", i)}).HashKey() || at.Key.Inspect() != fmt.Sprint("k", i) {
				t.Fatalf("size %d: At(%d) = %s", size, i, at.Key.Inspect())
			}
		}

		if !h.Delete(&String{Value: "k1"}) || h.Delete(&String{Value: "k1"}) {
			t.Fatalf("size %d: deleting k1 twice", size)
		}
		if h.Has((&String{Value: "k1"}).HashKey()) || h.Len() != size-1 {
			t.Fatalf("size %d: k1 is still there", size)
		}
		// The keys after the deleted one are still found
		last := fmt.Sprint("k", size-1)
		if pair, ok := h.Get((&String{Value: last}).HashKey()); !ok || pair.Key.Inspect() != last {
			t.Fatalf("size %d: %s not found after a delete", size, last)
		}
		if pairs := h.OrderedPairs(); pairs[0].Key.Inspect() != "k0" || (size > 2 && pairs[1].Key.Inspect() != "k2") {
			t.Fatalf("size %d: order after a delete: %v", size, pairs)
		}
	}
}

func TestOrderedPairsIsACopy(t *testing.T) {
	h := NewHash()
	h.Set(&String{Value: "a"}, &Integer{Value: 1})
	pairs := h.OrderedPairs()
	h.Set(&String{Value: "b"}, &Integer{Value: 2})
	pairs[0].Value = &Integer{Value: 9}
	if _, first := h.At(0); len(pairs) != 1 || first.Value.(*Integer).Value != 1 {
		t.Error("OrderedPairs shares its slice with the hash")
	}
}

func TestShapes(t *testing.T) {
	point := func(x, y int64) *Hash {
		h := NewHash()
		h.Set(&String{Value: "x"}, &Integer{Value: x})
		h.Set(&String{Value: "y"}, &Integer{Value: y})
		return h
	}
	a, b := point(1, 2), point(3, 4)
	if a.shape == nil || a.shape != b.shape {
		t.Fatal("objects with the same keys don't share a shape")
	}
	if pair, ok := b.Get((&String{Value: "y"}).HashKey()); !ok || pair.Value.(*Integer).Value != 4 {
		t.Errorf("y of b = %v", pair.Value)
	}

	// Updating a value keeps the shape; deleting a key makes a dictionary
	b.Set(&String{Value: "x"}, &Integer{Value: 9})
	if b.shape != a.shape {
		t.Error("updating a value changed the shape")
	}
	b.Delete(&String{Value: "x"})
	if b.shape != nil || b.Len() != 1 || !b.Has((&String{Value: "y"}).HashKey()) {
		t.Errorf("after a delete: shape %v, %d pairs", b.shape, b.Len())
	}

	// Many keys make a dictionary, keeping every pair in order
	big := NewHash()
	for i := 0; i < shapedUpTo+5; i++ {
		big.Set(&Integer{Value: int64(i)}, &Integer{Value: int64(i * 10)})
	}
	if big.shape != nil || big.Len() != shapedUpTo+5 {
		t.Fatalf("big: shape %v, %d pairs", big.shape, big.Len())
	}
	for i, pair := range big.OrderedPairs() {
		if pair.Key.(*Integer).Value != int64(i) || pair.Value.(*Integer).Value != int64(i*10) {
			t.Fatalf("pair %d is %v", i, pair)
		}
	}
}

func TestNewHashFromPairs(t *testing.T) {
	s := func(v string) Object { return &String{Value: v} }
	n := func(v int64) Object { return &Integer{Value: v} }
	var cache PropertyCache
	a, _ := NewHashFromPairs([]Object{s("x"), n(1), s("y"), n(2)}, &cache)
	b, _ := NewHashFromPairs([]Object{s("x"), n(3), s("y"), n(4)}, &cache)
	if a.shape != b.shape || cache.shape != a.shape {
		t.Fatal("a literal's objects don't share the shape in its cache")
	}
	// Keys computed at run time can differ between objects of one literal
	c, _ := NewHashFromPairs([]Object{s("y"), n(5), s("x"), n(6)}, &cache)
	if pairs := c.OrderedPairs(); pairs[0].Key.Inspect() != "y" || pairs[1].Value.Inspect() != "6" {
		t.Errorf("other keys: %v", pairs)
	}
	d, _ := NewHashFromPairs([]Object{s("x"), n(7), s("x"), n(8)}, &cache)
	if d.Len() != 1 || d.OrderedPairs()[0].Value.Inspect() != "8" {
		t.Errorf("repeated key: %v", d.OrderedPairs())
	}
	if _, bad := NewHashFromPairs([]Object{s("x"), n(1), &Array{}, n(2)}, &cache); bad != 2 {
		t.Errorf("bad key at %d, want 2", bad)
	}
}
