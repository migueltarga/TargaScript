package evaluator

import (
	"strconv"

	"github.com/migueltarga/TargaScript/object"
)

// CallFunction calls a TargaScript function, builtin or external value with
// args. It lets code outside the evaluator, such as a JavaScript event
// handler, call back into a script.
func CallFunction(fn object.Object, args []object.Object) object.Object {
	return applyFunction(fn, args)
}

// IsCallable reports whether obj can be called with CallFunction.
func IsCallable(obj object.Object) bool {
	return isCallable(obj)
}

// externalKey converts an index used on an external value to a property name.
func externalKey(index object.Object) (string, bool) {
	switch index := index.(type) {
	case *object.String:
		return index.Value, true
	case *object.Integer:
		return strconv.FormatInt(index.Value, 10), true
	default:
		return "", false
	}
}

// withPosition adds a source position to an error returned by an external
// value, which doesn't know where in the script it was used.
func withPosition(obj object.Object, line, column int) object.Object {
	if obj == nil {
		return NULL
	}
	if err, ok := obj.(*object.Error); ok && err.Line == 0 {
		err.Line, err.Column = line, column
	}
	return obj
}
