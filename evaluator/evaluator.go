package evaluator

import (
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/strfmt"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}

	// Where print writes; standard output outside the browser
	output = defaultOutput
)

// maxCallDepth limits recursion so runaway calls fail with an error
// instead of overflowing the Go stack.
const maxCallDepth = 10000

// SetOutput sets the writer print uses
func SetOutput(w io.Writer) {
	output = w
}

// Output returns the writer print uses.
func Output() io.Writer {
	return output
}

// FlushOutput writes out what the writer print uses keeps, if it keeps
// anything, before the program waits, reads input, runs a command or ends.
func FlushOutput() {
	if f, ok := output.(interface{ Flush() error }); ok {
		f.Flush()
	}
}

// increment returns current + 1 for "++" or current - 1 for "--".
func increment(current object.Object, operator string, line, column int) object.Object {
	delta := int64(1)
	if operator == "--" {
		delta = -1
	}

	switch current := current.(type) {
	case *object.Integer:
		value, ok := addInt(current.Value, delta)
		if !ok {
			return newError(line, column, "integer overflow")
		}
		return &object.Integer{Value: value}
	case *object.Float:
		return &object.Float{Value: current.Value + float64(delta)}
	default:
		return newError(line, column, "cannot use %q on %s", operator, typeName(current))
	}
}

// setProperty stores val in obj.name.
func setProperty(obj object.Object, propertyName string, val object.Object, line, column int) object.Object {
	if hash, ok := obj.(*object.Hash); ok {
		hash.Set(&object.String{Value: propertyName}, val)
		return val
	}

	if external, ok := obj.(object.External); ok {
		return withPosition(external.Set(propertyName, val), line, column)
	}

	return newError(line, column, "cannot set a property of %s", typeName(obj))
}

// setIndex stores val in container[index].
func setIndex(container, index, val object.Object, line, column int) object.Object {
	if array, ok := container.(*object.Array); ok {
		if idx, ok := index.(*object.Integer); ok {
			arrayLen := int64(len(array.Elements))
			arrayIdx := fromEnd(idx.Value, arrayLen)

			if arrayIdx < 0 || arrayIdx >= arrayLen {
				return newError(line, column, "index out of bounds: %d", idx.Value)
			}

			array.Elements[arrayIdx] = val
			return val
		}
		return newError(line, column, "array index must be an integer, got %s", typeName(index))
	}

	if hash, ok := container.(*object.Hash); ok {
		hashable, ok := index.(object.Hashable)
		if !ok {
			return newError(line, column, "%s can't be an object key", typeName(index))
		}

		hash.Set(hashable, val)
		return val
	}

	if external, ok := container.(object.External); ok {
		name, ok := externalKey(index)
		if !ok {
			return newError(line, column, "index must be a string or an integer, got %s", typeName(index))
		}
		return withPosition(external.Set(name, val), line, column)
	}

	return newError(line, column, "cannot use [] on %s", typeName(container))
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func evalPrefixExpression(operator string, right object.Object, line, column int) object.Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right, line, column)
	default:
		return newError(line, column, "cannot use %q on %s", operator, typeName(right))
	}
}

func evalInfixExpression(operator string, left, right object.Object, line, column int) object.Object {
	if left == nil || right == nil {
		return newError(line, column, "cannot evaluate infix expression: one of the operands is nil")
	}

	if operator == "in" {
		return evalInExpression(left, right, line, column)
	}

	if operator == ".." {
		start, startOk := left.(*object.Integer)
		end, endOk := right.(*object.Integer)
		if !startOk || !endOk {
			return newError(line, column, "range bounds must be integers, got %s and %s", typeName(left), typeName(right))
		}
		return integerRange(start.Value, end.Value, line, column)
	}

	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right, line, column)
	case left.Type() == object.FLOAT_OBJ && right.Type() == object.FLOAT_OBJ:
		return evalFloatInfixExpression(operator, left, right, line, column)
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.FLOAT_OBJ:
		return evalFloatInfixExpression(operator, toFloat(left), right, line, column)
	case left.Type() == object.FLOAT_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalFloatInfixExpression(operator, left, toFloat(right), line, column)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right, line, column)
	case left.Type() == object.STRING_OBJ && operator == "+":
		return object.Concat(left.(*object.String), objectToString(right))
	case left.Type() == object.STRING_OBJ && right.Type() == object.INTEGER_OBJ && operator == "*":
		return repeatString(left.(*object.String).Value, right.(*object.Integer).Value, line, column)
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.STRING_OBJ && operator == "*":
		return repeatString(right.(*object.String).Value, left.(*object.Integer).Value, line, column)
	case left.Type() == object.ARRAY_OBJ && right.Type() == object.ARRAY_OBJ && isOrdering(operator):
		return compareArrays(operator, left.(*object.Array), right.(*object.Array), line, column)
	case left.Type() == object.ARRAY_OBJ && right.Type() == object.ARRAY_OBJ && operator == "+":
		leftElements := left.(*object.Array).Elements
		rightElements := right.(*object.Array).Elements
		elements := make([]object.Object, 0, len(leftElements)+len(rightElements))
		elements = append(append(elements, leftElements...), rightElements...)
		return &object.Array{Elements: elements}
	case left.Type() == object.BOOLEAN_OBJ && right.Type() == object.BOOLEAN_OBJ:
		return evalBooleanInfixExpression(operator, left, right, line, column)
	case operator == "==":
		return nativeBoolToBooleanObject(valuesEqual(left, right, 0))
	case operator == "!=":
		return nativeBoolToBooleanObject(!valuesEqual(left, right, 0))
	case left.Type() != right.Type():
		return operatorError(line, column, operator, left, right)
	default:
		return operatorError(line, column, operator, left, right)
	}
}

func isOrdering(operator string) bool {
	return operator == "<" || operator == ">" || operator == "<=" || operator == ">="
}

// compareArrays orders arrays element by element, the first difference
// deciding, and a shorter array first when one starts the other: so
// sortBy(p => [p.city, p.name]) sorts by city, then by name.
func compareArrays(operator string, left, right *object.Array, line, column int) object.Object {
	for i := 0; i < len(left.Elements) && i < len(right.Elements); i++ {
		a, b := left.Elements[i], right.Elements[i]
		if valuesEqual(a, b, 0) {
			continue
		}
		less := evalInfixExpression("<", a, b, line, column)
		if isError(less) {
			return less
		}
		if less == TRUE {
			return nativeBoolToBooleanObject(operator == "<" || operator == "<=")
		}
		return nativeBoolToBooleanObject(operator == ">" || operator == ">=")
	}
	switch {
	case len(left.Elements) < len(right.Elements):
		return nativeBoolToBooleanObject(operator == "<" || operator == "<=")
	case len(left.Elements) > len(right.Elements):
		return nativeBoolToBooleanObject(operator == ">" || operator == ">=")
	}
	return nativeBoolToBooleanObject(operator == "<=" || operator == ">=")
}

// evalInExpression reports whether value is an element of an array, a key
// of an object or a part of a string.
func evalInExpression(value, collection object.Object, line, column int) object.Object {
	switch collection := collection.(type) {
	case *object.Array:
		for _, element := range collection.Elements {
			if valuesEqual(value, element, 0) {
				return TRUE
			}
		}
		return FALSE
	case *object.Hash:
		key, ok := value.(object.Hashable)
		if !ok {
			return FALSE
		}
		found := collection.Has(key.HashKey())
		return nativeBoolToBooleanObject(found)
	case *object.String:
		part, ok := value.(*object.String)
		if !ok {
			return newError(line, column, "cannot look for %s in a string, only for a string", typeName(value))
		}
		return nativeBoolToBooleanObject(strings.Contains(collection.Value, part.Value))
	}
	return newError(line, column, "cannot use \"in\" with %s, only with an array, an object or a string", typeName(collection))
}

// maxRepeatLength keeps "x" * n from asking for more memory than any
// program needs.
const maxRepeatLength = 1 << 28

func repeatString(text string, count int64, line, column int) object.Object {
	if count < 0 {
		return newError(line, column, "cannot repeat a string a negative number of times (%d)", count)
	}
	if text != "" && count > maxRepeatLength/int64(len(text)) {
		return newError(line, column, "the repeated string would be too long")
	}
	return &object.String{Value: strings.Repeat(text, int(count))}
}

func evalBangOperatorExpression(right object.Object) object.Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NULL:
		return TRUE
	default:
		return FALSE
	}
}

func evalMinusPrefixOperatorExpression(right object.Object, line, column int) object.Object {
	if right == nil {
		return newError(line, column, "cannot negate nil value")
	}

	switch right.Type() {
	case object.INTEGER_OBJ:
		value, ok := subInt(0, right.(*object.Integer).Value)
		if !ok {
			return newError(line, column, "integer overflow")
		}
		return &object.Integer{Value: value}
	case object.FLOAT_OBJ:
		value := right.(*object.Float).Value
		return &object.Float{Value: -value}
	default:
		return newError(line, column, "cannot use \"-\" on %s", typeName(right))
	}
}

func evalIntegerInfixExpression(operator string, left, right object.Object, line, column int) object.Object {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value

	switch operator {
	case "+", "-", "*", "/":
		if operator == "/" && rightVal == 0 {
			return newError(line, column, "division by zero")
		}
		value, ok := integerArithmetic[operator](leftVal, rightVal)
		if !ok {
			return newError(line, column, "integer overflow")
		}
		return &object.Integer{Value: value}
	case "%":
		if rightVal == 0 {
			return newError(line, column, "modulo by zero")
		}
		return &object.Integer{Value: leftVal % rightVal}
	case "^":
		return &object.Integer{Value: leftVal ^ rightVal}
	case "**":
		if rightVal < 0 {
			return &object.Float{Value: math.Pow(float64(leftVal), float64(rightVal))}
		}
		value, ok := powInt(leftVal, rightVal)
		if !ok {
			return newError(line, column, "integer overflow")
		}
		return &object.Integer{Value: value}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	default:
		return operatorError(line, column, operator, left, right)
	}
}

func evalFloatInfixExpression(operator string, left, right object.Object, line, column int) object.Object {
	leftVal := left.(*object.Float).Value
	rightVal := right.(*object.Float).Value

	switch operator {
	case "+":
		return &object.Float{Value: leftVal + rightVal}
	case "-":
		return &object.Float{Value: leftVal - rightVal}
	case "*":
		return &object.Float{Value: leftVal * rightVal}
	case "/":
		return &object.Float{Value: leftVal / rightVal}
	case "%":
		return &object.Float{Value: math.Mod(leftVal, rightVal)}
	case "**":
		return &object.Float{Value: math.Pow(leftVal, rightVal)}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	default:
		return operatorError(line, column, operator, left, right)
	}
}

func toFloat(obj object.Object) *object.Float {
	return &object.Float{Value: float64(obj.(*object.Integer).Value)}
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case NULL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		switch obj.Type() {
		case object.INTEGER_OBJ:
			return obj.(*object.Integer).Value != 0
		case object.FLOAT_OBJ:
			return obj.(*object.Float).Value != 0.0
		case object.STRING_OBJ:
			return len(obj.(*object.String).Value) > 0
		case object.ARRAY_OBJ:
			return len(obj.(*object.Array).Elements) > 0
		case object.HASH_OBJ:
			return obj.(*object.Hash).Len() > 0
		default:
			return true
		}
	}
}

func newError(line, column int, format string, a ...interface{}) *object.Error {
	return &object.Error{
		Message: strfmt.Sprintf(format, a...),
		Line:    line,
		Column:  column,
	}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

func applyFunction(fn object.Object, args []object.Object) object.Object {
	if fn == nil {
		return newError(0, 0, "function is nil")
	}

	switch fn := fn.(type) {
	case *object.Builtin:
		if fn == nil || fn.Fn == nil {
			return newError(0, 0, "builtin function is nil")
		}
		return fn.Fn(args...)
	case *object.Closure:
		return fn.Runner.CallClosure(fn, args)
	case object.External:
		return fn.Call(args)
	default:
		return newError(0, 0, "cannot call %s, it is not a function", typeName(fn))
	}
}

// applyFunctionAsync is applyFunction for a function that can pause at an
// await, in which case it returns the promise of its result.
func applyFunctionAsync(fn object.Object, args []object.Object) object.Object {
	if closure, ok := fn.(*object.Closure); ok {
		return closure.Runner.CallClosureAsync(closure, args)
	}
	return applyFunction(fn, args)
}

func isCallable(obj object.Object) bool {
	switch obj.(type) {
	case *object.Builtin, *object.Closure, object.External:
		return true
	default:
		return false
	}
}

func evalIndexExpression(left, index object.Object, line, column int) object.Object {
	if external, ok := left.(object.External); ok {
		name, ok := externalKey(index)
		if !ok {
			return newError(line, column, "index must be a string or an integer, got %s", typeName(index))
		}
		return withPosition(external.Get(name), line, column)
	}

	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return evalArrayIndexExpression(left, index)
	case left.Type() == object.HASH_OBJ:
		return evalHashIndexExpression(left, index, line, column)
	case left.Type() == object.STRING_OBJ && index.Type() == object.INTEGER_OBJ:
		chars := []rune(left.(*object.String).Value)
		idx := fromEnd(index.(*object.Integer).Value, int64(len(chars)))
		if idx < 0 || idx >= int64(len(chars)) {
			return NULL
		}
		return &object.String{Value: string(chars[idx])}
	default:
		return newError(line, column, "cannot use [] on %s", typeName(left))
	}
}

// fromEnd turns a negative index into one counted from the end, so -1 is
// the last element.
func fromEnd(index, length int64) int64 {
	if index < 0 {
		return index + length
	}
	return index
}

func evalArrayIndexExpression(array, index object.Object) object.Object {
	arrayObject := array.(*object.Array)
	idx := fromEnd(index.(*object.Integer).Value, int64(len(arrayObject.Elements)))
	max := int64(len(arrayObject.Elements) - 1)

	if idx < 0 || idx > max {
		return NULL
	}

	return arrayObject.Elements[idx]
}

func evalHashIndexExpression(hash, index object.Object, line, column int) object.Object {
	hashObject := hash.(*object.Hash)

	key, ok := index.(object.Hashable)
	if !ok {
		return newError(line, column, "%s can't be an object key", typeName(index))
	}

	pair, ok := hashObject.Get(key.HashKey())
	if !ok {
		return NULL
	}

	return pair.Value
}

// getProperty returns obj.name.
func getProperty(obj object.Object, propertyName string, line, column int) object.Object {
	switch obj := obj.(type) {
	case object.External:
		return withPosition(obj.Get(propertyName), line, column)
	case *object.Array:
		switch propertyName {
		case "length":
			return &object.Integer{Value: int64(len(obj.Elements))}
		default:
			return newError(line, column, "arrays have no property %q%s", propertyName, suggestion(propertyName, []string{"length"}))
		}
	case *object.String:
		switch propertyName {
		case "length":
			return &object.Integer{Value: int64(utf8.RuneCountInString(obj.Value))}
		default:
			return newError(line, column, "strings have no property %q%s", propertyName, suggestion(propertyName, []string{"length"}))
		}
	case *object.Hash:
		// An object's own properties come first, so data with a length
		// field, like {"length": 300}, reads as written
		if pair, ok := obj.Get((&object.String{Value: propertyName}).HashKey()); ok {
			return pair.Value
		}
		if propertyName == "length" {
			return &object.Integer{Value: int64(obj.Len())}
		}
		return NULL
	default:
		hint := ""
		if obj.Type() == object.NULL_OBJ {
			hint = strfmt.Sprintf(". To read it only when there's a value, write ?.%s", propertyName)
		}
		return newError(line, column, "cannot read property %q of %s%s", propertyName, typeName(obj), hint)
	}
}

func evalMethodCallExpression(obj object.Object, method string, args []object.Object, line, column int) object.Object {
	switch obj := obj.(type) {
	case object.External:
		if method == "new" {
			return withPosition(obj.New(args), line, column)
		}
		return withPosition(obj.CallMethod(method, args), line, column)
	case *object.String:
		switch method {
		case "trim":
			if len(args) != 0 {
				return newError(line, column, "wrong number of arguments for string.trim(): got %d, want 0", len(args))
			}
			return &object.String{Value: strings.TrimSpace(obj.Value)}

		case "lower":
			if len(args) != 0 {
				return newError(line, column, "wrong number of arguments for string.lower(): got %d, want 0", len(args))
			}
			return &object.String{Value: strings.ToLower(obj.Value)}

		case "upper":
			if len(args) != 0 {
				return newError(line, column, "wrong number of arguments for string.upper(): got %d, want 0", len(args))
			}
			return &object.String{Value: strings.ToUpper(obj.Value)}

		case "replace":
			if len(args) != 2 {
				return newError(line, column, "wrong number of arguments for string.replace(): got %d, want 2", len(args))
			}

			oldStr, ok := args[0].(*object.String)
			if !ok {
				return newError(line, column, "first argument to string.replace() must be a string, got %s", typeName(args[0]))
			}

			newStr, ok := args[1].(*object.String)
			if !ok {
				return newError(line, column, "second argument to string.replace() must be a string, got %s", typeName(args[1]))
			}

			return &object.String{Value: strings.Replace(obj.Value, oldStr.Value, newStr.Value, -1)}

		case "split":
			if len(args) != 1 {
				return newError(line, column, "wrong number of arguments for string.split(): got %d, want 1", len(args))
			}

			delimiter, ok := args[0].(*object.String)
			if !ok {
				return newError(line, column, "argument to string.split() must be a string, got %s", typeName(args[0]))
			}

			parts := strings.Split(obj.Value, delimiter.Value)
			elements := make([]object.Object, len(parts))
			for i, part := range parts {
				elements[i] = &object.String{Value: part}
			}

			return &object.Array{Elements: elements}

		case "contains", "indexOf", "startsWith", "endsWith":
			if len(args) != 1 {
				return newError(line, column, "wrong number of arguments for string.%s(): got %d, want 1", method, len(args))
			}
			sub, ok := args[0].(*object.String)
			if !ok {
				return newError(line, column, "argument to string.%s() must be a string, got %s", method, typeName(args[0]))
			}

			switch method {
			case "contains":
				return nativeBoolToBooleanObject(strings.Contains(obj.Value, sub.Value))
			case "startsWith":
				return nativeBoolToBooleanObject(strings.HasPrefix(obj.Value, sub.Value))
			case "endsWith":
				return nativeBoolToBooleanObject(strings.HasSuffix(obj.Value, sub.Value))
			default:
				// Index in characters, like string indexing and length
				byteIndex := strings.Index(obj.Value, sub.Value)
				if byteIndex < 0 {
					return &object.Integer{Value: -1}
				}
				return &object.Integer{Value: int64(utf8.RuneCountInString(obj.Value[:byteIndex]))}
			}

		default:
			if fn, ok := moreStringMethods[method]; ok {
				return withPosition(fn(obj.Value, args), line, column)
			}
			return newError(line, column, "strings have no method %q%s", method, hint(method, otherLanguageStringMethods, stringMethods))
		}
	case *object.Array:
		switch method {
		case "first":
			if len(args) != 0 {
				return newError(line, column, "wrong number of arguments for array.first(): got %d, want 0", len(args))
			}
			if len(obj.Elements) > 0 {
				return obj.Elements[0]
			}
			return NULL
		case "last":
			if len(args) != 0 {
				return newError(line, column, "wrong number of arguments for array.last(): got %d, want 0", len(args))
			}
			length := len(obj.Elements)
			if length > 0 {
				return obj.Elements[length-1]
			}
			return NULL
		case "insert":
			if len(args) != 1 {
				return newError(line, column, "wrong number of arguments for array.insert(): got %d, want 1", len(args))
			}

			// Appends in place, like index assignment, and returns the same array
			obj.Elements = append(obj.Elements, args[0])
			return obj
		case "rest":
			if len(args) != 0 {
				return newError(line, column, "wrong number of arguments for array.rest(): got %d, want 0", len(args))
			}

			if len(obj.Elements) == 0 {
				return &object.Array{Elements: []object.Object{}}
			}
			newElements := make([]object.Object, len(obj.Elements)-1)
			copy(newElements, obj.Elements[1:])
			return &object.Array{Elements: newElements}
		case "join":
			if len(args) != 1 {
				return newError(line, column, "wrong number of arguments for array.join(): got %d, want 1", len(args))
			}

			separator, ok := args[0].(*object.String)
			if !ok {
				return newError(line, column, "argument to array.join() must be a string, got %s", typeName(args[0]))
			}

			strs := make([]string, len(obj.Elements))
			for i, elem := range obj.Elements {
				strs[i] = elem.Inspect()
			}

			return &object.String{Value: strings.Join(strs, separator.Value)}
		case "map":
			if len(args) != 1 {
				return newError(line, column, "wrong number of arguments for array.map(): got %d, want 1", len(args))
			}

			fn := args[0]
			if !isCallable(fn) {
				return newError(line, column, "argument to array.map() must be a function, got %s", typeName(fn))
			}

			result := make([]object.Object, 0, len(obj.Elements))
			cb := newCallback(fn)
			for i, elem := range obj.Elements {
				// A function that awaits gives a promise, for all()
				mappedValue := applyFunctionAsync(fn, cb.with(elem, i))
				if isError(mappedValue) {
					return mappedValue
				}
				result = append(result, mappedValue)
			}

			return &object.Array{Elements: result}
		case "filter":
			if len(args) != 1 {
				return newError(line, column, "wrong number of arguments for array.filter(): got %d, want 1", len(args))
			}

			fn := args[0]
			if !isCallable(fn) {
				return newError(line, column, "argument to array.filter() must be a function, got %s", typeName(fn))
			}

			result := make([]object.Object, 0)
			cb := newCallback(fn)
			for i, elem := range obj.Elements {
				predicate := applyFunction(fn, cb.with(elem, i))
				if isError(predicate) {
					return predicate
				}

				if isTruthy(predicate) {
					result = append(result, elem)
				}
			}

			return &object.Array{Elements: result}
		case "reduce":
			if len(args) != 2 {
				return newError(line, column, "wrong number of arguments for array.reduce(): got %d, want 2", len(args))
			}

			fn := args[0]
			if !isCallable(fn) {
				return newError(line, column, "first argument to array.reduce() must be a function, got %s", typeName(fn))
			}

			initialValue := args[1]
			accumulator := initialValue

			cb := newCallback(fn)
			for i, elem := range obj.Elements {
				reduceArgs := cb.pair(accumulator, elem)
				if wantsArgs(fn, 3) {
					reduceArgs = append(reduceArgs, &object.Integer{Value: int64(i)})
				}
				accumulator = applyFunction(fn, reduceArgs)
				if isError(accumulator) {
					return accumulator
				}
			}

			return accumulator
		default:
			if result, ok := evalArraySearchMethod(obj, method, args, line, column); ok {
				return result
			}
			if result, ok := evalMoreArrayMethod(obj, method, args, line, column); ok {
				return result
			}
			if result, ok := evalArrayChangeMethod(obj, method, args, line, column); ok {
				return result
			}
			if result, ok := evalArrayShapeMethod(obj, method, args, line, column); ok {
				return result
			}
			return newError(line, column, "arrays have no method %q%s", method, hint(method, otherLanguageArrayMethods, arrayMethods))
		}
	case *object.Hash:
		// An object's own functions come before the methods every object has
		if pair, ok := obj.Get((&object.String{Value: method}).HashKey()); ok && isCallable(pair.Value) {
			return withPosition(applyFunction(pair.Value, args), line, column)
		}
		switch method {
		case "keys":
			if len(args) != 0 {
				return newError(line, column, "wrong number of arguments for object.keys(): got %d, want 0", len(args))
			}

			keys := []object.Object{}
			for _, pair := range obj.OrderedPairs() {
				keys = append(keys, pair.Key)
			}

			return &object.Array{Elements: keys}

		case "values":
			if len(args) != 0 {
				return newError(line, column, "wrong number of arguments for object.values(): got %d, want 0", len(args))
			}

			values := []object.Object{}
			for _, pair := range obj.OrderedPairs() {
				values = append(values, pair.Value)
			}

			return &object.Array{Elements: values}

		case "has":
			if len(args) != 1 {
				return newError(line, column, "wrong number of arguments for object.has(): got %d, want 1", len(args))
			}

			hashable, ok := args[0].(object.Hashable)
			if !ok {
				return newError(line, column, "argument to `has` must be hashable, got %s", typeName(args[0]))
			}

			exists := obj.Has(hashable.HashKey())
			return nativeBoolToBooleanObject(exists)

		case "length":
			if len(args) != 0 {
				return newError(line, column, "wrong number of arguments for object.length(): got %d, want 0", len(args))
			}

			return &object.Integer{Value: int64(obj.Len())}

		default:
			key := &object.String{Value: method}
			hashKey := key.HashKey()

			if pair, ok := obj.Get(hashKey); ok {
				if isCallable(pair.Value) {
					return withPosition(applyFunction(pair.Value, args), line, column)
				} else {
					return newError(line, column, "property %q is %s, not a function", method, typeName(pair.Value))
				}
			}

			// Methods an object's own functions don't override
			if fn, ok := moreHashMethods[method]; ok {
				return withPosition(fn(obj, args), line, column)
			}

			// The object's own functions are suggested before the methods
			own := []string{}
			for _, pair := range obj.OrderedPairs() {
				if name, ok := pair.Key.(*object.String); ok && isCallable(pair.Value) {
					own = append(own, name.Value)
				}
			}
			message := suggestion(method, own)
			if message == "" {
				message = hint(method, otherLanguageHashMethods, append([]string{}, hashMethods...))
			}
			return newError(line, column, "object has no method %q%s", method, message)
		}
	default:
		hint := ""
		if obj.Type() == object.NULL_OBJ {
			hint = strfmt.Sprintf(". To call it only when there's a value, write ?.%s()", method)
		}
		return newError(line, column, "cannot call method %q on %s%s", method, typeName(obj), hint)
	}
}

// maxRangeSize limits the arrays built by ranges so a typo like 1..1e12
// fails with an error instead of exhausting memory.
const maxRangeSize = 10_000_000

// integerRange returns the array [start, start+1, ..., end], or an empty
// array when start > end.
func integerRange(start, end int64, line, column int) object.Object {
	if start > end {
		return &object.Array{Elements: []object.Object{}}
	}
	if end-start >= maxRangeSize {
		return newError(line, column, "range too large: %d..%d (max %d elements)", start, end, maxRangeSize)
	}

	elements := make([]object.Object, 0, end-start+1)
	for i := start; i <= end; i++ {
		elements = append(elements, &object.Integer{Value: i})
	}
	return &object.Array{Elements: elements}
}

// printValues prints values separated by spaces, followed by a newline.
func printValues(args []object.Object) {
	values := make([]string, len(args))
	for i, arg := range args {
		if arg == nil {
			values[i] = "nil"
		} else {
			values[i] = objectToString(arg)
		}
	}

	io.WriteString(output, strings.Join(values, " ")+"\n")
}

func objectToString(obj object.Object) string {
	if obj == nil {
		return "nil"
	}

	switch obj.Type() {
	case object.INTEGER_OBJ:
		return strconv.FormatInt(obj.(*object.Integer).Value, 10)
	case object.FLOAT_OBJ:
		return obj.Inspect()
	case object.BOOLEAN_OBJ:
		return strconv.FormatBool(obj.(*object.Boolean).Value)
	case object.STRING_OBJ:
		return obj.(*object.String).Value
	case object.NULL_OBJ:
		return "null"
	case object.ARRAY_OBJ, object.HASH_OBJ, object.FUNCTION_OBJ, object.BUILTIN_OBJ:
		return obj.Inspect()
	default:
		return obj.Inspect()
	}
}

func evalBooleanInfixExpression(operator string, left, right object.Object, line, column int) object.Object {
	leftVal := left.(*object.Boolean).Value
	rightVal := right.(*object.Boolean).Value

	switch operator {
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return operatorError(line, column, operator, left, right)
	}
}

func evalStringInfixExpression(operator string, left, right object.Object, line, column int) object.Object {
	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value

	switch operator {
	case "+":
		return object.Concat(left.(*object.String), rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	default:
		return operatorError(line, column, operator, left, right)
	}
}
