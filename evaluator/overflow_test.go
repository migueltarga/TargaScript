package evaluator

import (
	"math"
	"math/big"
	"testing"
)

func TestOverflowHelpers(t *testing.T) {
	values := []int64{math.MinInt64, math.MinInt64 + 1, -3, -1, 0, 1, 2, 3, math.MaxInt64 - 1, math.MaxInt64}
	checks := []struct {
		name string
		fn   func(a, b int64) (int64, bool)
		want func(z, x, y *big.Int) *big.Int
	}{
		{"+", addInt, (*big.Int).Add},
		{"-", subInt, (*big.Int).Sub},
		{"*", mulInt, (*big.Int).Mul},
		{"/", divInt, (*big.Int).Quo},
	}

	for _, a := range values {
		for _, b := range values {
			for _, c := range checks {
				if c.name == "/" && b == 0 {
					continue
				}
				exact := c.want(new(big.Int), big.NewInt(a), big.NewInt(b))
				got, ok := c.fn(a, b)
				if ok != exact.IsInt64() || ok && got != exact.Int64() {
					t.Errorf("%d %s %d: got %d ok=%v, want %s", a, c.name, b, got, ok, exact)
				}
			}
		}
	}
}
