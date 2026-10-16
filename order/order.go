// Package order sorts with one stable merge sort, which works on indexes
// so every kind of list shares its code: a generic sort is compiled again
// for each type, which makes the WebAssembly build larger.
package order

// Stable returns the indexes 0 to n-1 ordered by compare, which is
// negative, zero or positive as the element at a goes before, with or after
// the one at b. Equal elements keep their order. It calls compare about
// n·log2(n) times, and skips merging halves already in order.
func Stable(n int, compare func(a, b int) int) []int {
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	if n > 1 {
		mergeSort(order, make([]int, n), compare)
	}
	return order
}

// Strings sorts s.
func Strings(s []string) {
	sorted := Stable(len(s), func(a, b int) int {
		switch {
		case s[a] < s[b]:
			return -1
		case s[a] > s[b]:
			return 1
		}
		return 0
	})
	copy(s, Permute(s, sorted))
}

// Permute returns the elements of s in the order of the indexes.
func Permute[T any](s []T, indexes []int) []T {
	out := make([]T, len(indexes))
	for i, j := range indexes {
		out[i] = s[j]
	}
	return out
}

func mergeSort(xs, buf []int, compare func(a, b int) int) {
	n := len(xs)
	if n <= 6 {
		// Insertion sort, for short runs
		for i := 1; i < n; i++ {
			for j := i; j > 0 && compare(xs[j-1], xs[j]) > 0; j-- {
				xs[j-1], xs[j] = xs[j], xs[j-1]
			}
		}
		return
	}
	mid := n / 2
	mergeSort(xs[:mid], buf[:mid], compare)
	mergeSort(xs[mid:], buf[mid:], compare)
	if compare(xs[mid-1], xs[mid]) <= 0 {
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
