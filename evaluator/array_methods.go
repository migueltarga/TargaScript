package evaluator

import (
	"cmp"
	"slices"

	"github.com/migueltarga/TargaScript/object"
)

// evalArraySearchMethod handles the array methods that search, slice or
// reorder elements. ok is false when method isn't one of them.
func evalArraySearchMethod(arr *object.Array, method string, args []object.Object, line, column int) (result object.Object, ok bool) {
	switch method {
	case "contains", "indexOf":
		if len(args) != 1 {
			return newError(line, column, "wrong number of arguments for array.%s(): got %d, want 1", method, len(args)), true
		}
		index := int64(-1)
		for i, element := range arr.Elements {
			if objectsEqual(element, args[0]) {
				index = int64(i)
				break
			}
		}
		if method == "contains" {
			return nativeBoolToBooleanObject(index >= 0), true
		}
		return &object.Integer{Value: index}, true

	case "find":
		if len(args) != 1 || !isCallable(args[0]) {
			return newError(line, column, "array.find() takes one FUNCTION argument"), true
		}
		cb := newCallback(args[0])
		for i, element := range arr.Elements {
			matched := applyFunction(args[0], cb.with(element, i))
			if isError(matched) {
				return matched, true
			}
			if isTruthy(matched) {
				return element, true
			}
		}
		return NULL, true

	case "slice":
		if len(args) < 1 || len(args) > 2 {
			return newError(line, column, "wrong number of arguments for array.slice(): got %d, want 1 or 2", len(args)), true
		}
		length := int64(len(arr.Elements))
		bounds := []int64{0, length}
		for i, arg := range args {
			n, isInt := arg.(*object.Integer)
			if !isInt {
				return newError(line, column, "arguments to array.slice() must be an integer, got %s", typeName(arg)), true
			}
			bounds[i] = clampSliceIndex(n.Value, length)
		}
		start, end := bounds[0], bounds[1]
		if start > end {
			start = end
		}
		elements := make([]object.Object, end-start)
		copy(elements, arr.Elements[start:end])
		return &object.Array{Elements: elements}, true

	case "reverse":
		if len(args) != 0 {
			return newError(line, column, "wrong number of arguments for array.reverse(): got %d, want 0", len(args)), true
		}
		for i, j := 0, len(arr.Elements)-1; i < j; i, j = i+1, j-1 {
			arr.Elements[i], arr.Elements[j] = arr.Elements[j], arr.Elements[i]
		}
		return arr, true

	case "sort":
		if len(args) > 1 || (len(args) == 1 && !isCallable(args[0])) {
			return newError(line, column, "array.sort() takes an optional FUNCTION argument"), true
		}
		if err := sortArray(arr, args, line, column); err != nil {
			return err, true
		}
		return arr, true
	}

	return nil, false
}

// clampSliceIndex turns a slice() index into a position in [0, length].
// Negative indexes count from the end.
func clampSliceIndex(index, length int64) int64 {
	if index < 0 {
		index += length
	}
	if index < 0 {
		return 0
	}
	if index > length {
		return length
	}
	return index
}

// sortArray sorts in place. With a compare function, its result decides the
// order like in JavaScript: negative if a comes first, positive if b does.
// Without one, elements must all be numbers or all be strings.
func sortArray(arr *object.Array, args []object.Object, line, column int) *object.Error {
	if len(args) == 0 {
		order, err := sortedOrder(arr.Elements, line, column)
		if err == nil {
			arr.Elements = inOrder(arr.Elements, order)
		}
		return err
	}

	var sortErr *object.Error
	cb := newCallback(args[0])
	slices.SortStableFunc(arr.Elements, func(a, b object.Object) int {
		if sortErr != nil {
			return 0
		}
		switch result := applyFunction(args[0], cb.pair(a, b)).(type) {
		case *object.Error:
			sortErr = result
		case *object.Integer:
			return cmp.Compare(result.Value, 0)
		case *object.Float:
			return cmp.Compare(result.Value, 0)
		default:
			sortErr = newError(line, column, "array.sort() compare function must return a number, got %s", typeName(result))
		}
		return 0
	})
	return sortErr
}

func objectsEqual(a, b object.Object) bool {
	return valuesEqual(a, b, 0)
}

// maxEqualDepth stops comparing arrays and objects nested this deep, which
// only happens when they contain themselves.
const maxEqualDepth = 100

// valuesEqual reports whether two values are equal: numbers by value, so 1
// equals 1.0, strings and booleans by value, arrays and objects by their
// contents, and other values, like functions, only when they are the same.
func valuesEqual(a, b object.Object, depth int) bool {
	if a == b {
		return true
	}
	if depth > maxEqualDepth {
		return false
	}

	switch a := a.(type) {
	case *object.Array:
		b, ok := b.(*object.Array)
		if !ok || len(a.Elements) != len(b.Elements) {
			return false
		}
		for i := range a.Elements {
			if !valuesEqual(a.Elements[i], b.Elements[i], depth+1) {
				return false
			}
		}
		return true

	case *object.Hash:
		b, ok := b.(*object.Hash)
		if !ok || a.Len() != b.Len() {
			return false
		}
		for i := range a.Len() {
			key, pair := a.At(i)
			other, ok := b.Get(key)
			if !ok || !valuesEqual(pair.Value, other.Value, depth+1) {
				return false
			}
		}
		return true

	case *object.Integer, *object.Float:
		switch b.(type) {
		case *object.Integer, *object.Float:
			return evalInfixExpression("==", a, b, 0, 0) == TRUE
		}
	case *object.String:
		b, ok := b.(*object.String)
		return ok && a.Value == b.Value
	case *object.Boolean:
		b, ok := b.(*object.Boolean)
		return ok && a.Value == b.Value
	}
	return false
}

// callbackArgs are the arguments an array method passes its function: the
// element, and its position too when the function takes a second
// parameter, as in list.map((item, i) => ...). Functions with one
// parameter, and builtins like print, get only the element.
func callbackArgs(fn object.Object, element object.Object, index int) []object.Object {
	if wantsArgs(fn, 2) {
		return []object.Object{element, &object.Integer{Value: int64(index)}}
	}
	return []object.Object{element}
}

// callback makes the arguments of the calls of a function given to an
// array method. A TargaScript function copies its arguments, so they go in
// the same slice every call.
type callback struct {
	fn   object.Object
	args []object.Object
}

func newCallback(fn object.Object) *callback {
	c := &callback{fn: fn}
	if _, ok := fn.(*object.Closure); ok {
		c.args = make([]object.Object, 3)
	}
	return c
}

// with returns the arguments for element at index.
func (c *callback) with(element object.Object, index int) []object.Object {
	if c.args == nil {
		return callbackArgs(c.fn, element, index)
	}
	c.args[0] = element
	if wantsArgs(c.fn, 2) {
		c.args[1] = &object.Integer{Value: int64(index)}
		return c.args[:2]
	}
	return c.args[:1]
}

// pair returns the arguments a and b, for reduce and sort.
func (c *callback) pair(a, b object.Object) []object.Object {
	if c.args == nil {
		return []object.Object{a, b}
	}
	c.args[0], c.args[1] = a, b
	return c.args[:2]
}

// wantsArgs reports whether fn is a TargaScript function that takes n
// arguments or more.
func wantsArgs(fn object.Object, n int) bool {
	cl, ok := fn.(*object.Closure)
	return ok && (cl.Fn.NumParams >= n || cl.Fn.Variadic)
}
