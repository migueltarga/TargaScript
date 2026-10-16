package order

import (
	"math/rand"
	"slices"
	"sort"
	"testing"
)

// Stable orders like a stable sort, for every size and many repeats
func TestStable(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for _, n := range []int{0, 1, 2, 5, 6, 7, 13, 100, 1000} {
		for _, keys := range []int{1, 3, n + 1} {
			k := make([]int, n)
			for i := range k {
				k[i] = r.Intn(keys)
			}
			want := make([]int, n)
			for i := range want {
				want[i] = i
			}
			sort.SliceStable(want, func(a, b int) bool { return k[want[a]] < k[want[b]] })
			got := Stable(n, func(a, b int) int { return k[a] - k[b] })
			if !slices.Equal(got, want) {
				t.Fatalf("n=%d keys=%d: got %v, want %v", n, keys, got, want)
			}
		}
	}
}

func TestStrings(t *testing.T) {
	s := []string{"pear", "apple", "fig", "apple", ""}
	Strings(s)
	if !slices.Equal(s, []string{"", "apple", "apple", "fig", "pear"}) {
		t.Errorf("got %v", s)
	}
}
