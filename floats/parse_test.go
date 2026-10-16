package floats

import (
	"math"
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

func jsNumber(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// parse accepts and reads what strconv.ParseFloat does
func TestParseLikeStrconv(t *testing.T) {
	inputs := []string{"1e-400", "1e400", "-1e400", "inf", "+Inf", "-infinity", "nan", "+nan", "NaN", "1_000", "0x1p-2",
		"0x1.8p1", "0x_1p0", " 1", "1.", ".5", "1e", "1e+", "0x10", "Infinity", "infin", "1E5", "-0", "00012", "0x1P+3",
		"", "+", "-", ".", "_1", "1_", "1__0", "0x", "0xp1", "0x1p", "1e1_0", "1.2.3", "0x1.fffffffffffff8p1023",
		"0x1p-1074", "0x1p-1075", "0x1p1024", "0x123456789abcdef0123p0", "1e99999999999", "123456789012345678901234567890",
		"4.9406564584124654e-324", "2.2250738585072014e-308", "1.7976931348623157e308", "1.7976931348623159e308"}
	r := rand.New(rand.NewSource(1))
	chars := "0123456789._eE+-pPxXabcfinty"
	for i := 0; i < 200000; i++ {
		var b strings.Builder
		for n := r.Intn(10) + 1; n > 0; n-- {
			b.WriteByte(chars[r.Intn(len(chars))])
		}
		inputs = append(inputs, b.String())
		inputs = append(inputs, strconv.FormatFloat(math.Float64frombits(r.Uint64()), 'g', -1, 64))
	}
	for _, s := range inputs {
		want, err := strconv.ParseFloat(s, 64)
		got, ok := parse(s, jsNumber)
		if ok != (err == nil) || ok && math.Float64bits(got) != math.Float64bits(want) && !(math.IsNaN(got) && math.IsNaN(want)) {
			t.Fatalf("parse(%q) = %v, %v; want %v, %v", s, got, ok, want, err)
		}
	}
}
