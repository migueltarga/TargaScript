package evaluator

import (
	"cmp"
	"math/rand"
	"slices"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

// mergeSort orders like a stable sort, for every size and many repeats
func TestMergeSortIsStable(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	byKey := func(a, b object.Object) int {
		return cmp.Compare(a.(*object.Array).Elements[0].(*object.Integer).Value, b.(*object.Array).Elements[0].(*object.Integer).Value)
	}
	for _, n := range []int{0, 1, 2, 5, 6, 7, 13, 100, 1000} {
		for _, keys := range []int{1, 3, n + 1} {
			xs := make([]object.Object, n)
			for i := range xs {
				xs[i] = &object.Array{Elements: []object.Object{&object.Integer{Value: int64(r.Intn(keys))}, &object.Integer{Value: int64(i)}}}
			}
			want := slices.Clone(xs)
			slices.SortStableFunc(want, byKey)
			mergeSort(xs, byKey, func() bool { return false })
			for i := range xs {
				if xs[i] != want[i] {
					t.Fatalf("n=%d keys=%d: differs at %d", n, keys, i)
				}
			}
		}
	}
}

// A sort that stops leaves the elements as they were
func TestMergeSortStops(t *testing.T) {
	xs := []object.Object{&object.Integer{Value: 3}, &object.Integer{Value: 1}, &object.Integer{Value: 2}}
	calls := 0
	mergeSort(xs, func(a, b object.Object) int { calls++; return 1 }, func() bool { return calls > 1 })
	if xs[0].(*object.Integer).Value != 3 || xs[1].(*object.Integer).Value != 1 || xs[2].(*object.Integer).Value != 2 {
		t.Errorf("got %v %v %v", xs[0], xs[1], xs[2])
	}
}
