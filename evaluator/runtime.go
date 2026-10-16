package evaluator

import (
	"github.com/migueltarga/TargaScript/diag"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/order"
)

// The operations below are shared with the virtual machine, so compiled code
// behaves exactly like evaluated code and reports the same errors.

// MaxCallDepth limits how deeply user function calls can nest.
const MaxCallDepth = maxCallDepth

// Infix applies a binary operator other than && and ||.
func Infix(operator string, left, right object.Object, line, column int) object.Object {
	return evalInfixExpression(operator, left, right, line, column)
}

// Prefix applies - or !.
func Prefix(operator string, right object.Object, line, column int) object.Object {
	return evalPrefixExpression(operator, right, line, column)
}

// Index returns left[index].
func Index(left, index object.Object, line, column int) object.Object {
	return evalIndexExpression(left, index, line, column)
}

// SetIndex stores val in container[index].
func SetIndex(container, index, val object.Object, line, column int) object.Object {
	return setIndex(container, index, val, line, column)
}

// GetProperty returns obj.name.
func GetProperty(obj object.Object, name string, line, column int) object.Object {
	return getProperty(obj, name, line, column)
}

// SetProperty stores val in obj.name.
func SetProperty(obj object.Object, name string, val object.Object, line, column int) object.Object {
	return setProperty(obj, name, val, line, column)
}

// CallMethod calls obj.method(args).
func CallMethod(obj object.Object, method string, args []object.Object, line, column int) object.Object {
	return evalMethodCallExpression(obj, method, args, line, column)
}

// Increment returns value + 1 for "++" or value - 1 for "--".
func Increment(value object.Object, operator string, line, column int) object.Object {
	return increment(value, operator, line, column)
}

// IsTruthy reports whether a value counts as true in conditions.
func IsTruthy(obj object.Object) bool {
	return isTruthy(obj)
}

// Diagnostic describes a runtime error for diag, with its trace.
func Diagnostic(err *object.Error) diag.Diagnostic {
	d := diag.Diagnostic{Line: err.Line, Column: err.Column, Message: err.Message}
	for _, call := range err.Trace {
		d.Trace = append(d.Trace, diag.Call{Function: call.Function, Line: call.Line})
	}
	return d
}

// ToDisplayString converts a value to the text print and interpolation show.
func ToDisplayString(obj object.Object) string {
	return objectToString(obj)
}

// IntegerRange returns the array start..end.
func IntegerRange(start, end int64, line, column int) object.Object {
	return integerRange(start, end, line, column)
}

// Builtin returns the builtin function with the given name.
func Builtin(name string) (*object.Builtin, bool) {
	builtin, ok := builtins[name]
	return builtin, ok
}

// BuiltinWithHost binds asynchronous builtins to one runtime. Other builtins
// keep their shared definitions.
func BuiltinWithHost(name string, h Host) (*object.Builtin, bool) {
	switch name {
	case "sleep":
		return &object.Builtin{Name: name, Fn: func(args ...object.Object) object.Object {
			return sleepWithHost(h, args...)
		}}, true
	case "fetch":
		return &object.Builtin{Name: name, Fn: func(args ...object.Object) object.Object {
			return fetchWithHost(h, args...)
		}}, true
	default:
		return Builtin(name)
	}
}

// BuiltinNames returns the names of the builtin functions, sorted.
func BuiltinNames() []string {
	names := make([]string, 0, len(builtins))
	for name := range builtins {
		names = append(names, name)
	}
	order.Strings(names)
	return names
}

// TypeName describes the type of a value for error messages, with an
// article: "an integer", "a string".
func TypeName(obj object.Object) string {
	return typeName(obj)
}

// NewError creates an error at a position.
func NewError(line, column int, format string, a ...interface{}) *object.Error {
	return newError(line, column, format, a...)
}

// SortByKeys sorts arr in place by keys, one for each element, as sortBy
// does, and returns it, or the error of keys that can't be compared.
func SortByKeys(arr *object.Array, keys []object.Object) object.Object {
	order, err := sortedOrder(keys, 0, 0)
	if err != nil {
		return err
	}
	arr.Elements = inOrder(arr.Elements, order)
	return arr
}
