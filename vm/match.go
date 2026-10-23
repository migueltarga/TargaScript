package vm

import (
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/object"
)

// matchPattern reports whether value matches pattern. values are what the
// pattern compares with, and the values its names take go in binds, by
// their index.
func matchPattern(pattern *object.Pattern, value object.Object, values, binds []object.Object) bool {
	switch pattern.Kind {
	case object.PatternAny:
		return true

	case object.PatternBind:
		binds[pattern.Index] = value
		return true

	case object.PatternValue:
		expected := values[pattern.Index]
		if n, ok := value.(*object.Integer); ok {
			if m, ok := expected.(*object.Integer); ok {
				return n.Value == m.Value
			}
		}
		return evaluator.Equal(value, expected)

	case object.PatternRange:
		// Values that can't be compared with the ends don't match
		above := evaluator.Infix(">=", value, values[pattern.Index], 0, 0)
		below := evaluator.Infix("<=", value, values[pattern.Index+1], 0, 0)
		return object.Same(above, TRUE) && object.Same(below, TRUE)

	case object.PatternArray:
		array, ok := value.(*object.Array)
		if !ok || len(array.Elements) < len(pattern.Parts) {
			return false
		}
		if pattern.Rest == nil && len(array.Elements) != len(pattern.Parts) {
			return false
		}
		for i, part := range pattern.Parts {
			if !matchPattern(part, array.Elements[i], values, binds) {
				return false
			}
		}
		if pattern.Rest != nil && pattern.Rest.Kind == object.PatternBind {
			binds[pattern.Rest.Index] = &object.Array{Elements: append([]object.Object(nil), array.Elements[len(pattern.Parts):]...)}
		}
		return true

	case object.PatternObject:
		hash, ok := value.(*object.Hash)
		if !ok {
			return false
		}
		for i, part := range pattern.Parts {
			pair, found := hash.Get(pattern.Keys.Elements[i].(*object.String).HashKey())
			if !found || !matchPattern(part, pair.Value, values, binds) {
				return false
			}
		}
		if pattern.Rest != nil && pattern.Rest.Kind == object.PatternBind {
			binds[pattern.Rest.Index] = without(hash, pattern.Keys)
		}
		return true
	}
	return false
}

// without returns a copy of an object without the keys in names, Strings.
func without(from *object.Hash, names *object.Array) *object.Hash {
	rest := object.NewHash()
pairs:
	for _, pair := range from.OrderedPairs() {
		for _, name := range names.Elements {
			if pair.Key.(object.Hashable).HashKey() == name.(*object.String).HashKey() {
				continue pairs
			}
		}
		rest.Set(pair.Key.(object.Hashable), pair.Value)
	}
	return rest
}
