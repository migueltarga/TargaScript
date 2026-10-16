package evaluator

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/migueltarga/TargaScript/object"
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
			return &object.Error{Message: fmt.Sprintf("%s takes an array or two or more numbers, got %s", name, typeName(args[0]))}
		}
		// Like list.max(), an empty array has none
		if len(array.Elements) == 0 {
			return NULL
		}
		values = array.Elements
	} else if len(args) == 0 {
		return &object.Error{Message: fmt.Sprintf("wrong number of arguments for %s. got=0, want an array or two or more numbers", name)}
	}

	var best object.Object
	bestValue := 0.0
	for _, value := range values {
		n, ok := number(value)
		if !ok {
			return &object.Error{Message: fmt.Sprintf("arguments to `%s` must be numbers, got %s", name, typeName(value))}
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
		return &object.Error{Message: fmt.Sprintf("wrong number of arguments for round. got=%d, want=1 or 2", len(args))}
	}
	value, ok := number(args[0])
	if !ok {
		return &object.Error{Message: fmt.Sprintf("argument to `round` must be a number, got %s", typeName(args[0]))}
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
			return &object.Error{Message: fmt.Sprintf("random takes nothing, an array or two integers, got %s", typeName(args[0]))}
		}
		if len(array.Elements) == 0 {
			return NULL
		}
		return array.Elements[rand.IntN(len(array.Elements))]
	case 2:
		low, lowOk := args[0].(*object.Integer)
		high, highOk := args[1].(*object.Integer)
		if !lowOk || !highOk {
			return &object.Error{Message: fmt.Sprintf("random(a, b) takes two integers, got %s and %s", typeName(args[0]), typeName(args[1]))}
		}
		if low.Value > high.Value {
			return &object.Error{Message: fmt.Sprintf("random(a, b) needs a <= b, got %d and %d", low.Value, high.Value)}
		}
		span := uint64(high.Value - low.Value)
		if span == math.MaxUint64 {
			return &object.Integer{Value: int64(rand.Uint64())}
		}
		return &object.Integer{Value: low.Value + int64(rand.Uint64N(span+1))}
	}
	return &object.Error{Message: fmt.Sprintf("wrong number of arguments for random. got=%d, want 0, 1 or 2", len(args))}
}

// rangeBuiltin returns the integers from start to end, both included,
// counting by step, which can be negative.
func rangeBuiltin(args ...object.Object) object.Object {
	if len(args) < 2 || len(args) > 3 {
		return &object.Error{Message: fmt.Sprintf("wrong number of arguments for range. got=%d, want=2 or 3", len(args))}
	}
	values := make([]int64, len(args))
	for i, arg := range args {
		integer, ok := arg.(*object.Integer)
		if !ok {
			return &object.Error{Message: fmt.Sprintf("arguments to range must be integers, got %s", typeName(arg))}
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
			return &object.Error{Message: fmt.Sprintf("range too large (max %d elements)", maxRangeSize)}
		}
		elements = append(elements, &object.Integer{Value: i})
		// Stop before i overflows
		if (step > 0 && i > math.MaxInt64-step) || (step < 0 && i < math.MinInt64-step) {
			break
		}
	}
	return &object.Array{Elements: elements}
}
