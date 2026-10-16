package floats

import (
	"math"
	"math/rand"
	"strconv"
	"testing"
)

// The digits JavaScript's toExponential gives, which strconv gives too
func jsShortest(f float64) string { return strconv.FormatFloat(f, 'e', -1, 64) }
func jsExact(f float64, d int) string {
	if f >= 1<<53 {
		return strconv.FormatFloat(f, 'f', 0, 64)
	}
	return strconv.FormatFloat(f, 'e', d, 64)
}
func layout(f float64, fmt byte, prec int) string { return format(f, fmt, prec, jsShortest, jsExact) }

func values(r *rand.Rand) []float64 {
	fs := []float64{0, math.Copysign(0, -1), 1, -1, 0.5, 1.5, 2.5, 0.125, 0.375, 1e21, 1e20, 123456789, 1e-7, 1e-5, 0.0001, 1e100, 1e-300,
		math.MaxFloat64, math.SmallestNonzeroFloat64, 5e-324, 9.999999, 99.995, 0.045, 1234.5678, 0.1, 0.2, 0.3, 1.0 / 3,
		math.Inf(1), math.Inf(-1), math.NaN(), 999999.5, 9.5, 0.95, 1e15 + 0.5}
	for i := 0; i < 20000; i++ {
		switch i % 4 {
		case 0:
			fs = append(fs, math.Float64frombits(r.Uint64()))
		case 1:
			fs = append(fs, float64(r.Int63n(1_000_000))/float64([]int{1, 10, 100, 1000, 8, 16}[r.Intn(6)]))
		case 2:
			fs = append(fs, r.NormFloat64()*math.Pow(10, float64(r.Intn(40)-20)))
		default:
			fs = append(fs, float64(r.Int63n(100000))+0.5)
		}
	}
	return fs
}

// format lays digits out like strconv.FormatFloat, for every format and
// precision the code uses
func TestFormatLikeStrconv(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	checked := 0
	for _, f := range values(r) {
		for _, fmt := range []byte{'e', 'E', 'f', 'g', 'G'} {
			for _, prec := range []int{-1, 0, 1, 2, 3, 5, 6, 10, 15, 17, 20, 30} {
				want := strconv.FormatFloat(f, fmt, prec, 64)
				if got := layout(f, fmt, prec); got != want {
					t.Fatalf("format(%v, %c, %d) = %q, want %q", f, fmt, prec, got, want)
				}
				checked++
			}
		}
	}
	t.Logf("%d formats checked", checked)
}
