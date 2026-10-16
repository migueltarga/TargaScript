package evaluator

import (
	"github.com/migueltarga/TargaScript/object"
)

func init() {
	arrayMethods = append(arrayMethods, "chunk", "zip", "toObject")
}

// evalArrayShapeMethod handles the array methods that split, pair or
// convert an array. ok is false when method isn't one of them.
func evalArrayShapeMethod(arr *object.Array, method string, args []object.Object, line, column int) (result object.Object, ok bool) {
	switch method {
	case "chunk":
		if len(args) != 1 {
			return wrongArgs(line, column, "array.chunk()", len(args), "1"), true
		}
		size, isInt := args[0].(*object.Integer)
		if !isInt || size.Value < 1 {
			return errorText(line, column, "the size for chunk must be a positive integer, got "+args[0].Inspect()), true
		}
		chunks := []object.Object{}
		for start := 0; start < len(arr.Elements); {
			end := start + int(min(size.Value, int64(len(arr.Elements)-start)))
			chunks = append(chunks, &object.Array{Elements: append([]object.Object{}, arr.Elements[start:end]...)})
			start = end
		}
		return &object.Array{Elements: chunks}, true

	case "zip":
		if len(args) != 1 {
			return wrongArgs(line, column, "array.zip()", len(args), "1"), true
		}
		other, isArray := args[0].(*object.Array)
		if !isArray {
			return errorText(line, column, "zip needs an array, got "+typeName(args[0])), true
		}
		length := min(len(arr.Elements), len(other.Elements))
		pairs := make([]object.Object, length)
		for i := range length {
			pair := object.NewArray(2)
			pair.Elements[0], pair.Elements[1] = arr.Elements[i], other.Elements[i]
			pairs[i] = pair
		}
		return &object.Array{Elements: pairs}, true

	case "toObject":
		if len(args) != 0 {
			return wrongArgs(line, column, "array.toObject()", len(args), "0"), true
		}
		hash := object.NewHash()
		for _, element := range arr.Elements {
			pair, isArray := element.(*object.Array)
			if !isArray || len(pair.Elements) != 2 {
				return errorText(line, column, "toObject needs [key, value] pairs, got "+element.Inspect()), true
			}
			key, hashable := pair.Elements[0].(object.Hashable)
			if !hashable {
				return errorText(line, column, typeName(pair.Elements[0])+" can't be an object key"), true
			}
			hash.Set(key, pair.Elements[1])
		}
		return hash, true
	}
	return nil, false
}
