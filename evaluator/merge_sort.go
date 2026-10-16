package evaluator

import "github.com/migueltarga/TargaScript/object"

// mergeSort sorts xs stably with compare, which a function of the program
// runs: it calls compare about n·log2(n) times, fewer than
// slices.SortStableFunc, and skips merging halves already in order. It
// stops early when stop reports true, leaving xs as it was.
func mergeSort(xs []object.Object, compare func(a, b object.Object) int, stop func() bool) {
	if len(xs) < 2 {
		return
	}
	sorted := append([]object.Object(nil), xs...)
	buf := make([]object.Object, len(xs))
	mergeSortInto(sorted, buf, compare, stop)
	if !stop() {
		copy(xs, sorted)
	}
}

func mergeSortInto(xs, buf []object.Object, compare func(a, b object.Object) int, stop func() bool) {
	n := len(xs)
	if n <= 6 {
		// Insertion sort, for short runs
		for i := 1; i < n && !stop(); i++ {
			for j := i; j > 0 && compare(xs[j-1], xs[j]) > 0; j-- {
				xs[j-1], xs[j] = xs[j], xs[j-1]
			}
		}
		return
	}
	mid := n / 2
	mergeSortInto(xs[:mid], buf[:mid], compare, stop)
	mergeSortInto(xs[mid:], buf[mid:], compare, stop)
	if stop() || compare(xs[mid-1], xs[mid]) <= 0 {
		// The halves are in order already
		return
	}
	copy(buf, xs[:mid])
	left, right, out := buf[:mid], xs[mid:], xs[:0]
	for len(left) > 0 && len(right) > 0 {
		// Equal elements keep their order: the left one goes first
		if compare(left[0], right[0]) <= 0 {
			out = append(out, left[0])
			left = left[1:]
		} else {
			out = append(out, right[0])
			right = right[1:]
		}
	}
	// What's left of right is already in place
	copy(xs[len(out):], left)
}
