package evaluator

import (
	"sort"

	"github.com/migueltarga/TargaScript/object"
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

// ToDisplayString converts a value to the text print and interpolation show.
func ToDisplayString(obj object.Object) string {
	return objectToString(obj)
}

// Print writes values like the print statement.
func Print(values []object.Object) {
	printValues(values)
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

// BuiltinNames returns the names of the builtin functions, sorted.
func BuiltinNames() []string {
	names := make([]string, 0, len(builtins))
	for name := range builtins {
		names = append(names, name)
	}
	sort.Strings(names)
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
