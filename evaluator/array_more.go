package evaluator

import (
	"sort"

	"github.com/migueltarga/TargaScript/object"
)

func init() {
	arrayMethods = append(arrayMethods, "some", "every", "count", "sum", "min", "max", "unique", "flatten", "groupBy", "sortBy")
}

// evalMoreArrayMethod handles the array methods that summarize, group or
// reshape elements. ok is false when method isn't one of them.
func evalMoreArrayMethod(arr *object.Array, method string, args []object.Object, line, column int) (result object.Object, ok bool) {
	switch method {
	case "some", "every":
		fn, err := callbackArg(method, args, line, column)
		if err != nil {
			return err, true
		}
		for _, element := range arr.Elements {
			result := applyFunction(fn, []object.Object{element})
			if isError(result) {
				return withPosition(result, line, column), true
			}
			if isTruthy(result) == (method == "some") {
				return nativeBoolToBooleanObject(method == "some"), true
			}
		}
		return nativeBoolToBooleanObject(method == "every"), true

	case "count":
		if len(args) != 1 {
			return newError(line, column, "wrong number of arguments for array.count(): got %d, want 1", len(args)), true
		}
		count := int64(0)
		for _, element := range arr.Elements {
			if isCallable(args[0]) {
				result := applyFunction(args[0], []object.Object{element})
				if isError(result) {
					return withPosition(result, line, column), true
				}
				if isTruthy(result) {
					count++
				}
			} else if objectsEqual(element, args[0]) {
				count++
			}
		}
		return &object.Integer{Value: count}, true

	case "sum":
		if len(args) != 0 {
			return newError(line, column, "wrong number of arguments for array.sum(): got %d, want 0", len(args)), true
		}
		var total object.Object = &object.Integer{Value: 0}
		for _, element := range arr.Elements {
			switch element.(type) {
			case *object.Integer, *object.Float:
			default:
				return newError(line, column, "array.sum() can only add numbers, got %s", typeName(element)), true
			}
			total = evalInfixExpression("+", total, element, line, column)
			if isError(total) {
				return total, true
			}
		}
		return total, true

	case "min", "max":
		if len(args) != 0 {
			return newError(line, column, "wrong number of arguments for array.%s(): got %d, want 0", method, len(args)), true
		}
		if len(arr.Elements) == 0 {
			return NULL, true
		}
		operator := "<"
		if method == "max" {
			operator = ">"
		}
		best := arr.Elements[0]
		for _, element := range arr.Elements[1:] {
			better := evalInfixExpression(operator, element, best, line, column)
			if errObj, isErr := better.(*object.Error); isErr {
				return newError(line, column, "cannot find the %s: %s", method, errObj.Message), true
			}
			if better == TRUE {
				best = element
			}
		}
		return best, true

	case "unique":
		if len(args) != 0 {
			return newError(line, column, "wrong number of arguments for array.unique(): got %d, want 0", len(args)), true
		}
		unique := []object.Object{}
		seen := map[object.HashKey]bool{}
		for _, element := range arr.Elements {
			if key, ok := element.(object.Hashable); ok {
				if seen[key.HashKey()] {
					continue
				}
				seen[key.HashKey()] = true
				unique = append(unique, element)
				continue
			}
			duplicate := false
			for _, kept := range unique {
				if objectsEqual(kept, element) {
					duplicate = true
					break
				}
			}
			if !duplicate {
				unique = append(unique, element)
			}
		}
		return &object.Array{Elements: unique}, true

	case "flatten":
		depth := int64(1)
		if len(args) > 1 {
			return newError(line, column, "wrong number of arguments for array.flatten(): got %d, want 0 or 1", len(args)), true
		}
		if len(args) == 1 {
			n, ok := args[0].(*object.Integer)
			if !ok {
				return newError(line, column, "depth for array.flatten() must be an integer, got %s", typeName(args[0])), true
			}
			depth = n.Value
		}
		return &object.Array{Elements: flatten(arr.Elements, depth)}, true

	case "groupBy":
		fn, err := callbackArg(method, args, line, column)
		if err != nil {
			return err, true
		}
		groups := object.NewHash()
		for _, element := range arr.Elements {
			key := applyFunction(fn, []object.Object{element})
			if isError(key) {
				return withPosition(key, line, column), true
			}
			hashable, ok := key.(object.Hashable)
			if !ok {
				return newError(line, column, "array.groupBy() function must return a string, number or boolean, got %s", typeName(key)), true
			}
			var group object.Object = NULL
			if pair, ok := groups.Get(hashable.HashKey()); ok {
				group = pair.Value
			}
			if group == NULL {
				group = &object.Array{Elements: []object.Object{}}
				groups.Set(hashable, group)
			}
			list := group.(*object.Array)
			list.Elements = append(list.Elements, element)
		}
		return groups, true

	case "sortBy":
		fn, err := callbackArg(method, args, line, column)
		if err != nil {
			return err, true
		}
		keys := make([]object.Object, len(arr.Elements))
		for i, element := range arr.Elements {
			keys[i] = applyFunction(fn, []object.Object{element})
			if isError(keys[i]) {
				return withPosition(keys[i], line, column), true
			}
		}
		indexes := make([]int, len(arr.Elements))
		for i := range indexes {
			indexes[i] = i
		}
		var sortErr *object.Error
		sort.SliceStable(indexes, func(a, b int) bool {
			if sortErr != nil {
				return false
			}
			lt := evalInfixExpression("<", keys[indexes[a]], keys[indexes[b]], line, column)
			if errObj, isErr := lt.(*object.Error); isErr {
				sortErr = newError(line, column, "cannot sort: %s", errObj.Message)
				return false
			}
			return lt == TRUE
		})
		if sortErr != nil {
			return sortErr, true
		}
		sorted := make([]object.Object, len(indexes))
		for i, index := range indexes {
			sorted[i] = arr.Elements[index]
		}
		arr.Elements = sorted
		return arr, true
	}
	return nil, false
}

func callbackArg(method string, args []object.Object, line, column int) (object.Object, *object.Error) {
	if len(args) != 1 || !isCallable(args[0]) {
		return nil, newError(line, column, "array.%s() takes a function", method)
	}
	return args[0], nil
}

// flatten puts the elements of nested arrays in place of the arrays, depth
// levels deep.
func flatten(elements []object.Object, depth int64) []object.Object {
	flat := []object.Object{}
	for _, element := range elements {
		if inner, ok := element.(*object.Array); ok && depth > 0 {
			flat = append(flat, flatten(inner.Elements, depth-1)...)
		} else {
			flat = append(flat, element)
		}
	}
	return flat
}
