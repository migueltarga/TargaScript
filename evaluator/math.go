package evaluator

import (
	"math"
	"math/rand/v2"
	"strconv"

	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/strfmt"
)

func init() {
	builtins["min"] = &object.Builtin{Name: "min", Fn: func(args ...object.Object) object.Object {
		return extreme("min", args, func(a, b float64) bool { return a < b })
	}}
	builtins["max"] = &object.Builtin{Name: "max", Fn: func(args ...object.Object) object.Object {
		return extreme("max", args, func(a, b float64) bool { return a > b })
	}}
	builtins["round"] = &object.Builtin{Name: "round", Fn: roundBuiltin}
	builtins["random"] = &object.Builtin{Name: "random", Fn: randomBuiltin}
	builtins["range"] = &object.Builtin{Name: "range", Fn: rangeBuiltin}
}

// number returns the value of an integer or a float.
func number(obj object.Object) (float64, bool) {
	switch obj := obj.(type) {
	case *object.Integer:
		return float64(obj.Value), true
	case *object.Float:
		return obj.Value, true
	}
	return 0, false
}

// extreme returns the smallest or largest of its arguments, or of the
// elements of an array given alone, keeping integers as integers.
func extreme(name string, args []object.Object, better func(a, b float64) bool) object.Object {
	values := args
	if len(args) == 1 {
		array, ok := args[0].(*object.Array)
		if !ok {
			return &object.Error{Message: strfmt.Sprintf("%s takes an array or two or more numbers, got %s", name, typeName(args[0]))}
		}
		// Like list.max(), an empty array has none
		if len(array.Elements) == 0 {
			return NULL
		}
		values = array.Elements
	} else if len(args) == 0 {
		return wrongArgs(0, 0, name, 0, "an array or two or more numbers")
	}

	var best object.Object
	bestValue := 0.0
	for _, value := range values {
		n, ok := number(value)
		if !ok {
			return &object.Error{Message: strfmt.Sprintf("arguments to `%s` must be numbers, got %s", name, typeName(value))}
		}
		if best == nil || better(n, bestValue) {
			best, bestValue = value, n
		}
	}
	return best
}

// roundBuiltin rounds to the nearest integer, or to a number of decimal
// places, giving a float, when there's a second argument.
func roundBuiltin(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return wrongArgs(0, 0, "round", len(args), "1 or 2")
	}
	value, ok := number(args[0])
	if !ok {
		return &object.Error{Message: strfmt.Sprintf("argument to `round` must be a number, got %s", typeName(args[0]))}
	}

	if len(args) == 2 {
		digits, ok := args[1].(*object.Integer)
		if !ok || digits.Value < 0 || digits.Value > 15 {
			return &object.Error{Message: "the decimal places for `round` must be an integer from 0 to 15"}
		}
		scale := math.Pow(10, float64(digits.Value))
		return &object.Float{Value: math.Round(value*scale) / scale}
	}

	if integer, ok := args[0].(*object.Integer); ok {
		return integer
	}
	result, ok := floatToInt(math.Round(value))
	if !ok {
		return &object.Error{Message: "integer overflow"}
	}
	return &object.Integer{Value: result}
}

// randomBuiltin returns a float in [0, 1), an integer from a to b, or a
// random element of an array.
func randomBuiltin(args ...object.Object) object.Object {
	switch len(args) {
	case 0:
		return &object.Float{Value: rand.Float64()}
	case 1:
		array, ok := args[0].(*object.Array)
		if !ok {
			return &object.Error{Message: strfmt.Sprintf("random takes nothing, an array or two integers, got %s", typeName(args[0]))}
		}
		if len(array.Elements) == 0 {
			return NULL
		}
		return array.Elements[rand.IntN(len(array.Elements))]
	case 2:
		low, lowOk := args[0].(*object.Integer)
		high, highOk := args[1].(*object.Integer)
		if !lowOk || !highOk {
			return &object.Error{Message: strfmt.Sprintf("random(a, b) takes two integers, got %s and %s", typeName(args[0]), typeName(args[1]))}
		}
		if low.Value > high.Value {
			return &object.Error{Message: strfmt.Sprintf("random(a, b) needs a <= b, got %d and %d", low.Value, high.Value)}
		}
		span := uint64(high.Value - low.Value)
		if span == math.MaxUint64 {
			return &object.Integer{Value: int64(rand.Uint64())}
		}
		return &object.Integer{Value: low.Value + int64(rand.Uint64N(span+1))}
	}
	return wrongArgs(0, 0, "random", len(args), "0, 1 or 2")
}

// rangeBuiltin returns the integers from start to end, both included,
// counting by step, which can be negative.
func rangeBuiltin(args ...object.Object) object.Object {
	if len(args) < 2 || len(args) > 3 {
		return wrongArgs(0, 0, "range", len(args), "2 or 3")
	}
	values := make([]int64, len(args))
	for i, arg := range args {
		integer, ok := arg.(*object.Integer)
		if !ok {
			return &object.Error{Message: strfmt.Sprintf("arguments to range must be integers, got %s", typeName(arg))}
		}
		values[i] = integer.Value
	}
	start, end := values[0], values[1]
	if len(values) == 2 {
		return integerRange(start, end, 0, 0)
	}

	step := values[2]
	if step == 0 {
		return &object.Error{Message: "the step of range can't be 0"}
	}
	elements := []object.Object{}
	for i := start; (step > 0 && i <= end) || (step < 0 && i >= end); i += step {
		if len(elements) >= maxRangeSize {
			return &object.Error{Message: strfmt.Sprintf("range too large (max %d elements)", maxRangeSize)}
		}
		elements = append(elements, &object.Integer{Value: i})
		// Stop before i overflows
		if (step > 0 && i > math.MaxInt64-step) || (step < 0 && i < math.MinInt64-step) {
			break
		}
	}
	return &object.Array{Elements: elements}
}

// constants are values every program can read by name, unless it declares
// the name itself.
var constants = map[string]object.Object{
	"PI": &object.Float{Value: math.Pi},
	"E":  &object.Float{Value: math.E},
}

// Constant returns the value of a constant like PI.
func Constant(name string) (object.Object, bool) {
	value, ok := constants[name]
	return value, ok
}

// MathHost computes sin, log and the like. Go's versions are large in
// WebAssembly, so the browser build uses JavaScript's Math instead.
type MathHost interface {
	// Math applies the function name, like "sin" or "atan2", to args
	Math(name string, args ...float64) float64
}

var mathHost MathHost

// SetMathHost sets what sin, log and the like use.
func SetMathHost(host MathHost) {
	mathHost = host
}

// mathNames are the functions of MathHost. A list, not a map literal,
// because TinyGo turns each entry of a map literal into code.
var mathNames = []string{"sin", "cos", "tan", "asin", "acos", "atan", "log", "log10", "log2", "exp", "atan2", "hypot"}

// mathArity is how many numbers a function of MathHost takes.
func mathArity(name string) int {
	if name == "atan2" || name == "hypot" {
		return 2
	}
	return 1
}

func init() {
	for _, name := range mathNames {
		builtins[name] = &object.Builtin{Name: name, Fn: mathBuiltin(name)}
	}
	builtins["trunc"] = &object.Builtin{Name: "trunc", Fn: truncBuiltin}
	builtins["sign"] = &object.Builtin{Name: "sign", Fn: signBuiltin}
	builtins["clamp"] = &object.Builtin{Name: "clamp", Fn: clampBuiltin}
}

// mathBuiltin is the builtin for a function of MathHost.
func mathBuiltin(name string) func(args ...object.Object) object.Object {
	return func(args ...object.Object) object.Object {
		if len(args) != mathArity(name) {
			return wrongArgs(0, 0, name, len(args), strconv.Itoa(mathArity(name)))
		}
		values := make([]float64, len(args))
		for i, arg := range args {
			n, ok := number(arg)
			if !ok {
				return &object.Error{Message: strfmt.Sprintf("arguments to `%s` must be numbers, got %s", name, typeName(arg))}
			}
			values[i] = n
		}
		if mathHost == nil {
			return &object.Error{Message: name + " is not available here"}
		}
		return &object.Float{Value: mathHost.Math(name, values...)}
	}
}

// truncBuiltin is a number without its decimals, toward zero.
func truncBuiltin(args ...object.Object) object.Object {
	if len(args) != 1 {
		return wrongArgs(0, 0, "trunc", len(args), "1")
	}
	x, ok := number(args[0])
	if !ok {
		return &object.Error{Message: strfmt.Sprintf("argument to `trunc` must be a number, got %s", typeName(args[0]))}
	}
	return &object.Float{Value: math.Trunc(x)}
}

// signBuiltin is -1, 0 or 1, an integer, as the number is negative, zero or
// positive.
func signBuiltin(args ...object.Object) object.Object {
	if len(args) != 1 {
		return wrongArgs(0, 0, "sign", len(args), "1")
	}
	x, ok := number(args[0])
	if !ok {
		return &object.Error{Message: strfmt.Sprintf("argument to `sign` must be a number, got %s", typeName(args[0]))}
	}
	switch {
	case x < 0:
		return &object.Integer{Value: -1}
	case x > 0:
		return &object.Integer{Value: 1}
	}
	return &object.Integer{Value: 0}
}

// clampBuiltin keeps a number between a minimum and a maximum.
func clampBuiltin(args ...object.Object) object.Object {
	if len(args) != 3 {
		return wrongArgs(0, 0, "clamp", len(args), "3 (value, min, max)")
	}
	value, low, high := args[0], args[1], args[2]
	for _, arg := range args {
		if _, ok := number(arg); !ok {
			return &object.Error{Message: strfmt.Sprintf("arguments to `clamp` must be numbers, got %s", typeName(arg))}
		}
	}
	if result := extreme("max", []object.Object{value, low}, func(a, b float64) bool { return a > b }); result != nil {
		value = result
	}
	return extreme("min", []object.Object{value, high}, func(a, b float64) bool { return a < b })
}
