package evaluator

import (
	"fmt"

	"github.com/migueltarga/TargaScript/object"
)

func init() {
	hashMethods = append(hashMethods, "entries", "get", "remove")
	arrayMethods = append(arrayMethods, "pop", "shift", "unshift", "removeAt")
	builtins["format"] = &object.Builtin{Name: "format", Fn: format}
}

// moreHashMethods are object methods that an object's own function with the
// same name takes precedence over, so existing objects with a get or remove
// function keep working.
var moreHashMethods = map[string]func(h *object.Hash, args []object.Object) object.Object{
	"entries": func(h *object.Hash, args []object.Object) object.Object {
		if len(args) != 0 {
			return &object.Error{Message: fmt.Sprintf("wrong number of arguments for object.entries(): got %d, want 0", len(args))}
		}
		entries := []object.Object{}
		for _, pair := range h.OrderedPairs() {
			entries = append(entries, &object.Array{Elements: []object.Object{pair.Key, pair.Value}})
		}
		return &object.Array{Elements: entries}
	},
	"get": func(h *object.Hash, args []object.Object) object.Object {
		if len(args) != 1 && len(args) != 2 {
			return &object.Error{Message: fmt.Sprintf("wrong number of arguments for object.get(): got %d, want 1 or 2", len(args))}
		}
		key, ok := args[0].(object.Hashable)
		if !ok {
			return &object.Error{Message: fmt.Sprintf("unusable as object key: %s", typeName(args[0]))}
		}
		if pair, ok := h.Pairs[key.HashKey()]; ok {
			return pair.Value
		}
		if len(args) == 2 {
			return args[1]
		}
		return NULL
	},
	"remove": func(h *object.Hash, args []object.Object) object.Object {
		if len(args) != 1 {
			return &object.Error{Message: fmt.Sprintf("wrong number of arguments for object.remove(): got %d, want 1", len(args))}
		}
		key, ok := args[0].(object.Hashable)
		if !ok {
			return &object.Error{Message: fmt.Sprintf("unusable as object key: %s", typeName(args[0]))}
		}
		var value object.Object = NULL
		if pair, ok := h.Pairs[key.HashKey()]; ok {
			value = pair.Value
		}
		h.Delete(key)
		return value
	},
}

// evalArrayChangeMethod handles the array methods that add or remove
// elements at either end or at a position, changing the array.
func evalArrayChangeMethod(arr *object.Array, method string, args []object.Object, line, column int) (object.Object, bool) {
	wantArgs := func(n int) *object.Error {
		if len(args) != n {
			return newError(line, column, "wrong number of arguments for array.%s(): got %d, want %d", method, len(args), n)
		}
		return nil
	}

	switch method {
	case "pop", "shift":
		if err := wantArgs(0); err != nil {
			return err, true
		}
		if len(arr.Elements) == 0 {
			return NULL, true
		}
		if method == "pop" {
			last := arr.Elements[len(arr.Elements)-1]
			arr.Elements = arr.Elements[:len(arr.Elements)-1]
			return last, true
		}
		first := arr.Elements[0]
		arr.Elements = append([]object.Object{}, arr.Elements[1:]...)
		return first, true

	case "unshift":
		if err := wantArgs(1); err != nil {
			return err, true
		}
		arr.Elements = append([]object.Object{args[0]}, arr.Elements...)
		return arr, true

	case "removeAt":
		if err := wantArgs(1); err != nil {
			return err, true
		}
		index, ok := args[0].(*object.Integer)
		if !ok {
			return newError(line, column, "position for array.removeAt() must be an integer, got %s", typeName(args[0])), true
		}
		i := index.Value
		if i < 0 {
			i += int64(len(arr.Elements))
		}
		if i < 0 || i >= int64(len(arr.Elements)) {
			return newError(line, column, "position %d is outside the array, which has %d elements", index.Value, len(arr.Elements)), true
		}
		removed := arr.Elements[i]
		arr.Elements = append(arr.Elements[:i], arr.Elements[i+1:]...)
		return removed, true
	}
	return nil, false
}

// format formats values like printf, and returns the text instead of
// printing it.
func format(args ...object.Object) object.Object {
	if len(args) < 1 {
		return &object.Error{Message: "wrong number of arguments for format. got=0, want at least 1"}
	}
	text, ok := args[0].(*object.String)
	if !ok {
		return &object.Error{Message: fmt.Sprintf("first argument to format must be a string, got %s", typeName(args[0]))}
	}
	formatted, err := formatPrintf(text.Value, args[1:])
	if err != nil {
		return err
	}
	return &object.String{Value: formatted}
}
