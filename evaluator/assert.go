package evaluator

import (
	"strconv"

	"github.com/migueltarga/TargaScript/object"
)

func init() {
	builtins["assert"] = &object.Builtin{Name: "assert", Fn: assertBuiltin}
	builtins["assertEqual"] = &object.Builtin{Name: "assertEqual", Fn: assertEqualBuiltin}
}

// assertBuiltin fails when its first argument is falsy, with the message
// in the second when there is one.
func assertBuiltin(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return wrongArgs(0, 0, "assert", len(args), "1 or 2")
	}
	if isTruthy(args[0]) {
		return NULL
	}
	if len(args) == 2 {
		return &object.Error{Message: "assertion failed: " + objectToString(args[1])}
	}
	return &object.Error{Message: "assertion failed"}
}

// assertEqualBuiltin fails when its arguments aren't equal, as == compares
// them, showing both.
func assertEqualBuiltin(args ...object.Object) object.Object {
	if len(args) < 2 || len(args) > 3 {
		return wrongArgs(0, 0, "assertEqual", len(args), "2 or 3")
	}
	if valuesEqual(args[0], args[1], 0) {
		return NULL
	}
	message := "expected " + quoted(args[1]) + ", got " + quoted(args[0])
	if len(args) == 3 {
		message = objectToString(args[2]) + ": " + message
	}
	return &object.Error{Message: "assertion failed: " + message}
}

// quoted shows a value as code would write it, with strings in quotes.
func quoted(obj object.Object) string {
	if str, ok := obj.(*object.String); ok {
		return strconv.Quote(str.Value)
	}
	return obj.Inspect()
}
