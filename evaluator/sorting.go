package evaluator

import (
	"cmp"
	"slices"
	"strings"

	"github.com/migueltarga/TargaScript/object"
)

// sortedOrder returns the indexes of keys in the order < puts them, keeping
// equal keys in their order. Keys that are all integers or all strings are
// compared directly, which is many times faster than going through <.
func sortedOrder(keys []object.Object, line, column int) ([]int, *object.Error) {
	order := make([]int, len(keys))
	for i := range order {
		order[i] = i
	}

	if ints, ok := allInts(keys); ok {
		slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(ints[a], ints[b]) })
		return order, nil
	}
	if texts, ok := allStrings(keys); ok {
		slices.SortStableFunc(order, func(a, b int) int { return strings.Compare(texts[a], texts[b]) })
		return order, nil
	}

	var sortErr *object.Error
	slices.SortStableFunc(order, func(a, b int) int {
		if sortErr != nil {
			return 0
		}
		for result, pair := range [2][2]object.Object{{keys[a], keys[b]}, {keys[b], keys[a]}} {
			lt := evalInfixExpression("<", pair[0], pair[1], line, column)
			if err, ok := lt.(*object.Error); ok {
				sortErr = newError(line, column, "cannot sort: %s", err.Message)
				return 0
			}
			if lt == TRUE {
				return result*2 - 1
			}
		}
		return 0
	})
	return order, sortErr
}

func allInts(values []object.Object) ([]int64, bool) {
	ints := make([]int64, len(values))
	for i, v := range values {
		n, ok := v.(*object.Integer)
		if !ok {
			return nil, false
		}
		ints[i] = n.Value
	}
	return ints, true
}

func allStrings(values []object.Object) ([]string, bool) {
	texts := make([]string, len(values))
	for i, v := range values {
		s, ok := v.(*object.String)
		if !ok {
			return nil, false
		}
		texts[i] = s.Value
	}
	return texts, true
}

// inOrder returns the elements in an order from sortedOrder.
func inOrder(elements []object.Object, order []int) []object.Object {
	sorted := make([]object.Object, len(order))
	for i, index := range order {
		sorted[i] = elements[index]
	}
	return sorted
}
