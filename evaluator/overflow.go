package evaluator

import "math"

// integerArithmetic holds the integer operators that can overflow. Each
// returns the result and whether it fits in an int64.
var integerArithmetic = map[string]func(a, b int64) (int64, bool){
	"+": addInt,
	"-": subInt,
	"*": mulInt,
	"/": divInt,
}

func addInt(a, b int64) (int64, bool) {
	c := a + b
	return c, (c > a) == (b > 0)
}

func subInt(a, b int64) (int64, bool) {
	c := a - b
	return c, (c < a) == (b > 0)
}

func mulInt(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	c := a * b
	if (a == -1 && b == math.MinInt64) || (b == -1 && a == math.MinInt64) {
		return c, false
	}
	return c, c/b == a
}

func divInt(a, b int64) (int64, bool) {
	if a == math.MinInt64 && b == -1 {
		return a, false
	}
	return a / b, true
}

// floatToInt converts v to an int64, reporting false when v is NaN or
// outside the int64 range.
func floatToInt(v float64) (int64, bool) {
	if math.IsNaN(v) || v < math.MinInt64 || v >= math.MaxInt64 {
		return 0, false
	}
	return int64(v), true
}
