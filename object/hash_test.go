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
	if len(pairs) != 1 || h.pairs[0].Value.(*Integer).Value != 1 {
		t.Error("OrderedPairs shares its slice with the hash")
	}
}
