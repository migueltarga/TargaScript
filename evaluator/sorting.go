package evaluator

import (
	"cmp"
	"strings"

	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/order"
)

// sortedOrder returns the indexes of keys in the order < puts them, keeping
// equal keys in their order. Keys that are all integers or all strings are
// compared directly, which is many times faster than going through <.
func sortedOrder(keys []object.Object, line, column int) ([]int, *object.Error) {
	if ints, ok := allInts(keys); ok {
		return order.Stable(len(keys), func(a, b int) int { return cmp.Compare(ints[a], ints[b]) }), nil
	}
	if texts, ok := allStrings(keys); ok {
		return order.Stable(len(keys), func(a, b int) int { return strings.Compare(texts[a], texts[b]) }), nil
	}

	var sortErr *object.Error
	sorted := order.Stable(len(keys), func(a, b int) int {
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
	return sorted, sortErr
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
