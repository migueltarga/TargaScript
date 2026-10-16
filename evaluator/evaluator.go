package evaluator

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/migueltarga/TargaScript/object"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}

	// Default output to stdout
	output io.Writer = os.Stdout
)

// maxCallDepth limits recursion so runaway calls fail with an error
// instead of overflowing the Go stack.
const maxCallDepth = 10000

// SetOutput sets the writer to use for print statements
func SetOutput(w io.Writer) {
	output = w
}

// Output returns the writer print statements use.
func Output() io.Writer {
	return output
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
		return newError(line, column, "operator %s not supported for %s", operator, current.Type())
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

	return newError(line, column, "cannot assign to property of non-object: %s", obj.Type())
}

// setIndex stores val in container[index].
func setIndex(container, index, val object.Object, line, column int) object.Object {
	if array, ok := container.(*object.Array); ok {
		if idx, ok := index.(*object.Integer); ok {
			arrayIdx := idx.Value
			arrayLen := int64(len(array.Elements))

			if arrayIdx < 0 || arrayIdx >= arrayLen {
				return newError(line, column, "index out of bounds: %d", arrayIdx)
			}

			array.Elements[arrayIdx] = val
			return val
		}
		return newError(line, column, "array index must be an integer, got: %s", index.Type())
	}

	if hash, ok := container.(*object.Hash); ok {
		hashable, ok := index.(object.Hashable)
		if !ok {
			return newError(line, column, "unusable as hash key: %s", index.Type())
		}

		hash.Set(hashable, val)
		return val
	}

	if external, ok := container.(object.External); ok {
		name, ok := externalKey(index)
		if !ok {
			return newError(line, column, "index must be a string or an integer, got: %s", index.Type())
		}
		return withPosition(external.Set(name, val), line, column)
	}

	return newError(line, column, "index assignment not supported for: %s", container.Type())
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

	if operator == ".." {
		start, startOk := left.(*object.Integer)
		end, endOk := right.(*object.Integer)
		if !startOk || !endOk {
			return newError(line, column, "range bounds must be integers, got %s..%s", left.Type(), right.Type())
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
		leftVal := left.(*object.String).Value
		rightVal := objectToString(right)
		return &object.String{Value: leftVal + rightVal}
	case left.Type() == object.BOOLEAN_OBJ && right.Type() == object.BOOLEAN_OBJ:
		return evalBooleanInfixExpression(operator, left, right, line, column)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case left.Type() != right.Type():
		return operatorError(line, column, operator, left, right)
	default:
		return operatorError(line, column, operator, left, right)
	}
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
			return len(obj.(*object.Hash).Pairs) > 0
		default:
			return true
		}
	}
}

func newError(line, column int, format string, a ...interface{}) *object.Error {
	return &object.Error{
		Message: fmt.Sprintf(format, a...),
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
			return newError(line, column, "index must be a string or an integer, got: %s", index.Type())
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
		idx := index.(*object.Integer).Value
		if idx < 0 || idx >= int64(len(chars)) {
			return NULL
		}
		return &object.String{Value: string(chars[idx])}
	default:
		return newError(line, column, "cannot use [] on %s", typeName(left))
	}
}

func evalArrayIndexExpression(array, index object.Object) object.Object {
	arrayObject := array.(*object.Array)
	idx := index.(*object.Integer).Value
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
		return newError(line, column, "unusable as hash key: %s", index.Type())
	}

	pair, ok := hashObject.Pairs[key.HashKey()]
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

		if propertyName == "length" {
			return &object.Integer{Value: int64(len(obj.Pairs))}
		}

		key := &object.String{Value: propertyName}
		hashKey := key.HashKey()

		if pair, ok := obj.Pairs[hashKey]; ok {
			return pair.Value
		}
		return NULL
	default:
		return newError(line, column, "dot operator not supported for %s", obj.Type())
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
				return newError(line, column, "first argument to string.replace() must be STRING, got %s", args[0].Type())
			}

			newStr, ok := args[1].(*object.String)
			if !ok {
				return newError(line, column, "second argument to string.replace() must be STRING, got %s", args[1].Type())
			}

			return &object.String{Value: strings.Replace(obj.Value, oldStr.Value, newStr.Value, -1)}

		case "split":
			if len(args) != 1 {
				return newError(line, column, "wrong number of arguments for string.split(): got %d, want 1", len(args))
			}

			delimiter, ok := args[0].(*object.String)
			if !ok {
				return newError(line, column, "argument to string.split() must be STRING, got %s", args[0].Type())
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
				return newError(line, column, "argument to string.%s() must be STRING, got %s", method, args[0].Type())
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
			return newError(line, column, "strings have no method %q%s", method, suggestion(method, stringMethods))
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
				return newError(line, column, "argument to array.join() must be STRING, got %s", args[0].Type())
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
				return newError(line, column, "argument to array.map() must be FUNCTION, got %s", fn.Type())
			}

			result := make([]object.Object, 0, len(obj.Elements))
			for _, elem := range obj.Elements {
				mappedValue := applyFunction(fn, []object.Object{elem})
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
				return newError(line, column, "argument to array.filter() must be FUNCTION, got %s", fn.Type())
			}

			result := make([]object.Object, 0)
			for _, elem := range obj.Elements {
				predicate := applyFunction(fn, []object.Object{elem})
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
				return newError(line, column, "first argument to array.reduce() must be FUNCTION, got %s", fn.Type())
			}

			initialValue := args[1]
			accumulator := initialValue

			for _, elem := range obj.Elements {
				accumulator = applyFunction(fn, []object.Object{accumulator, elem})
				if isError(accumulator) {
					return accumulator
				}
			}

			return accumulator
		default:
			if result, ok := evalArraySearchMethod(obj, method, args, line, column); ok {
				return result
			}
			return newError(line, column, "arrays have no method %q%s", method, suggestion(method, arrayMethods))
		}
	case *object.Hash:
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
				return newError(line, column, "argument to `has` must be hashable, got %s", args[0].Type())
			}

			_, exists := obj.Pairs[hashable.HashKey()]
			return nativeBoolToBooleanObject(exists)

		case "length":
			if len(args) != 0 {
				return newError(line, column, "wrong number of arguments for object.length(): got %d, want 0", len(args))
			}

			return &object.Integer{Value: int64(len(obj.Pairs))}

		default:
			key := &object.String{Value: method}
			hashKey := key.HashKey()

			if pair, ok := obj.Pairs[hashKey]; ok {
				if isCallable(pair.Value) {
					return withPosition(applyFunction(pair.Value, args), line, column)
				} else {
					return newError(line, column, "property %q is %s, not a function", method, typeName(pair.Value))
				}
			}

			candidates := append([]string{}, hashMethods...)
			for _, pair := range obj.Pairs {
				if name, ok := pair.Key.(*object.String); ok && isCallable(pair.Value) {
					candidates = append(candidates, name.Value)
				}
			}
			return newError(line, column, "object has no method %q%s", method, suggestion(method, candidates))
		}
	default:
		return newError(line, column, "cannot call method %q on %s", method, typeName(obj))
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

	fmt.Fprintln(output, strings.Join(values, " "))
}

func objectToString(obj object.Object) string {
	if obj == nil {
		return "nil"
	}

	switch obj.Type() {
	case object.INTEGER_OBJ:
		return fmt.Sprintf("%d", obj.(*object.Integer).Value)
	case object.FLOAT_OBJ:
		return obj.Inspect()
	case object.BOOLEAN_OBJ:
		return fmt.Sprintf("%t", obj.(*object.Boolean).Value)
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
		return &object.String{Value: leftVal + rightVal}
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
