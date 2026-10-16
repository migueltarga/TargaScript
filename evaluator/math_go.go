//go:build !tinygo

package evaluator

import "math"

// goMath computes with Go's math package, outside the browser build.
type goMath struct{}

func (goMath) Math(name string, args ...float64) float64 {
	switch name {
	case "sin":
		return math.Sin(args[0])
	case "cos":
		return math.Cos(args[0])
	case "tan":
		return math.Tan(args[0])
	case "asin":
		return math.Asin(args[0])
	case "acos":
		return math.Acos(args[0])
	case "atan":
		return math.Atan(args[0])
	case "log":
		return math.Log(args[0])
	case "log10":
		return math.Log10(args[0])
	case "log2":
		return math.Log2(args[0])
	case "exp":
		return math.Exp(args[0])
	case "atan2":
		return math.Atan2(args[0], args[1])
	case "hypot":
		return math.Hypot(args[0], args[1])
	}
	return math.NaN()
}

func init() {
	SetMathHost(goMath{})
}
