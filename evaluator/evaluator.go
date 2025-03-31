package evaluator

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/object"
)

var (
	NULL     = &object.Null{}
	TRUE     = &object.Boolean{Value: true}
	FALSE    = &object.Boolean{Value: false}
	BREAK    = &object.Break{}
	CONTINUE = &object.Continue{}

	// Default output to stdout
	output io.Writer = os.Stdout
)

// SetOutput sets the writer to use for print statements
func SetOutput(w io.Writer) {
	output = w
}

// Eval evaluates the AST node and returns an object
func Eval(node ast.Node, env *object.Environment) object.Object {
	if node == nil {
		fmt.Fprintf(output, "WARNING: Eval received nil node\n")
		return NULL
	}

	switch node := node.(type) {
	// Statements
	case *ast.Program:
		return evalProgram(node.Statements, env)

	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	case *ast.RepeatStatement:
		return evalRepeatStatement(node, env)

	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}

	case *ast.BlockStatement:
		return evalBlockStatements(node.Statements, env)

	case *ast.LetStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		env.Set(node.Name.Value, val)
		return val

	case *ast.FunctionStatement:
		fn := &object.Function{
			Parameters: node.Parameters,
			Body:       node.Body,
			Env:        env,
			Name:       node.Name.Value,
		}
		env.Set(node.Name.Value, fn)
		return fn

	case *ast.BreakStatement:
		return BREAK

	case *ast.ContinueStatement:
		return CONTINUE

	// Expressions
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	case *ast.FloatLiteral:
		return &object.Float{Value: node.Value}

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)

	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right, node.Token.Line, node.Token.Column)

	case *ast.InfixExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}

		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}

		return evalInfixExpression(node.Operator, left, right, node.Token.Line, node.Token.Column)

	case *ast.IfExpression:
		return evalIfExpression(node, env)

	case *ast.Identifier:
		return evalIdentifier(node, env)

	case *ast.FunctionLiteral:
		params := node.Parameters
		body := node.Body
		name := ""
		if node.Name != nil {
			name = node.Name.Value
		}
		return &object.Function{Parameters: params, Body: body, Env: env, Name: name}

	case *ast.CallExpression:
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}
		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunction(function, args)

	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}

	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}
		return evalIndexExpression(left, index, node.Token.Line, node.Token.Column)

	case *ast.HashLiteral:
		return evalHashLiteral(node, env)

	case *ast.PrintStatement:
		return evalPrintExpression(node, env)

	case *ast.DotExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		return evalDotExpression(left, node, env)

	case *ast.MethodCallExpression:
		object := Eval(node.Object, env)
		if isError(object) {
			return object
		}
		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return evalMethodCallExpression(object, node.Method.Value, args, node.Token.Line, node.Token.Column)

	case *ast.AssignmentExpression:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}

		switch left := node.Left.(type) {
		case *ast.Identifier:
			if _, ok := env.Get(left.Value); !ok {
				return newError(node.Token.Line, node.Token.Column, "identifier not found: %s", left.Value)
			}
			env.Set(left.Value, val)
			return val

		case *ast.DotExpression:
			obj := Eval(left.Left, env)
			if isError(obj) {
				return obj
			}

			propertyName := left.Property.Value

			if hash, ok := obj.(*object.Hash); ok {
				key := &object.String{Value: propertyName}
				hashKey := key.HashKey()

				hash.Pairs[hashKey] = object.HashPair{Key: key, Value: val}
				return val
			}

			return newError(node.Token.Line, node.Token.Column, "cannot assign to property of non-object: %s", obj.Type())

		case *ast.IndexExpression:
			container := Eval(left.Left, env)
			if isError(container) {
				return container
			}

			index := Eval(left.Index, env)
			if isError(index) {
				return index
			}

			if array, ok := container.(*object.Array); ok {
				if idx, ok := index.(*object.Integer); ok {
					arrayIdx := idx.Value
					arrayLen := int64(len(array.Elements))

					if arrayIdx < 0 || arrayIdx >= arrayLen {
						return newError(node.Token.Line, node.Token.Column, "index out of bounds: %d", arrayIdx)
					}

					array.Elements[arrayIdx] = val
					return val
				}
				return newError(node.Token.Line, node.Token.Column, "array index must be an integer, got: %s", index.Type())
			}

			if hash, ok := container.(*object.Hash); ok {
				hashable, ok := index.(object.Hashable)
				if !ok {
					return newError(node.Token.Line, node.Token.Column, "unusable as hash key: %s", index.Type())
				}

				hash.Pairs[hashable.HashKey()] = object.HashPair{Key: index, Value: val}
				return val
			}

			return newError(node.Token.Line, node.Token.Column, "index assignment not supported for: %s", container.Type())

		default:
			return newError(node.Token.Line, node.Token.Column, "invalid assignment target: %T", node.Left)
		}
	}

	nodeType := fmt.Sprintf("%T", node)
	fmt.Fprintf(output, "WARNING: Unhandled node type: %s\n", nodeType)
	return NULL
}

func evalProgram(statements []ast.Statement, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range statements {
		result = Eval(statement, env)

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}
	}

	return result
}

func evalBlockStatements(statements []ast.Statement, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range statements {
		result = Eval(statement, env)

		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ || rt == object.BREAK_OBJ || rt == object.CONTINUE_OBJ {
				return result
			}
		}
	}

	if result == nil {
		fmt.Fprintf(output, "WARNING: evalBlockStatements returning NULL because result is nil\n")
		return NULL
	}

	return result
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
		return newError(line, column, "unknown operator: %s%s", operator, right.Type())
	}
}

func evalInfixExpression(operator string, left, right object.Object, line, column int) object.Object {
	if left == nil || right == nil {
		return newError(line, column, "cannot evaluate infix expression: one of the operands is nil")
	}

	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right, line, column)
	case left.Type() == object.FLOAT_OBJ && right.Type() == object.FLOAT_OBJ:
		return evalFloatInfixExpression(operator, left, right, line, column)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right, line, column)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case left.Type() != right.Type():
		return newError(line, column, "type mismatch: %s %s %s",
			left.Type(), operator, right.Type())
	default:
		return newError(line, column, "unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
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
		value := right.(*object.Integer).Value
		return &object.Integer{Value: -value}
	case object.FLOAT_OBJ:
		value := right.(*object.Float).Value
		return &object.Float{Value: -value}
	default:
		return newError(line, column, "unknown operator: -%s", right.Type())
	}
}

func evalIntegerInfixExpression(operator string, left, right object.Object, line, column int) object.Object {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value

	switch operator {
	case "+":
		return &object.Integer{Value: leftVal + rightVal}
	case "-":
		return &object.Integer{Value: leftVal - rightVal}
	case "*":
		return &object.Integer{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return newError(line, column, "division by zero")
		}
		return &object.Integer{Value: leftVal / rightVal}
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
	case "&&":
		if leftVal != 0 && rightVal != 0 {
			return TRUE
		}
		return FALSE
	case "||":
		if leftVal != 0 || rightVal != 0 {
			return TRUE
		}
		return FALSE
	default:
		return newError(line, column, "unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
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
		if rightVal == 0 {
			return newError(line, column, "division by zero")
		}
		return &object.Float{Value: leftVal / rightVal}
	case "%":
		if rightVal == 0 {
			return newError(line, column, "modulo by zero")
		}
		return &object.Float{Value: float64(int64(leftVal) % int64(rightVal))}
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
		return newError(line, column, "unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func evalStringInfixExpression(operator string, left, right object.Object, line, column int) object.Object {
	if operator != "+" {
		return newError(line, column, "unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}

	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value
	return &object.String{Value: leftVal + rightVal}
}

func evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	} else {
		return NULL
	}
}

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}

	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}

	return newError(node.Token.Line, node.Token.Column, "identifier not found: %s", node.Value)
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

func evalExpressions(exps []ast.Expression, env *object.Environment) []object.Object {
	var result []object.Object

	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

func applyFunction(fn object.Object, args []object.Object) object.Object {
	if fn == nil {
		return newError(0, 0, "function is nil")
	}

	switch fn := fn.(type) {
	case *object.Function:
		if fn == nil {
			return newError(0, 0, "function object is nil")
		}
		extendedEnv := extendFunctionEnv(fn, args)

		evaluated := Eval(fn.Body, extendedEnv)

		return unwrapReturnValue(evaluated)
	case *object.Builtin:
		if fn == nil || fn.Fn == nil {
			return newError(0, 0, "builtin function is nil")
		}
		return fn.Fn(args...)
	default:
		return newError(0, 0, "not a function: %s", fn.Type())
	}
}

func extendFunctionEnv(fn *object.Function, args []object.Object) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env)

	for paramIdx, param := range fn.Parameters {
		if paramIdx < len(args) {
			env.Set(param.Value, args[paramIdx])
		} else {
			env.Set(param.Value, NULL)
		}
	}

	return env
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}

	return obj
}

func evalIndexExpression(left, index object.Object, line, column int) object.Object {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return evalArrayIndexExpression(left, index)
	case left.Type() == object.HASH_OBJ:
		return evalHashIndexExpression(left, index, line, column)
	default:
		return newError(line, column, "index operator not supported: %s", left.Type())
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

func evalHashLiteral(node *ast.HashLiteral, env *object.Environment) object.Object {
	pairs := make(map[object.HashKey]object.HashPair)

	for _, pair := range node.Pairs {
		key := Eval(pair.Key, env)
		if isError(key) {
			return key
		}

		hashKey, ok := key.(object.Hashable)
		if !ok {
			return newError(node.Token.Line, node.Token.Column, "unusable as hash key: %s", key.Type())
		}

		value := Eval(pair.Value, env)
		if isError(value) {
			return value
		}

		hashed := hashKey.HashKey()
		pairs[hashed] = object.HashPair{Key: key, Value: value}
	}

	return &object.Hash{Pairs: pairs}
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

func evalDotExpression(obj object.Object, node *ast.DotExpression, env *object.Environment) object.Object {
	switch obj := obj.(type) {
	case *object.Array:
		propertyName := node.Property.Value
		switch propertyName {
		case "length":
			return &object.Integer{Value: int64(len(obj.Elements))}
		default:
			return newError(node.Token.Line, node.Token.Column, "array has no property '%s'", propertyName)
		}
	case *object.String:
		propertyName := node.Property.Value
		switch propertyName {
		case "length":
			return &object.Integer{Value: int64(len(obj.Value))}
		default:
			return newError(node.Token.Line, node.Token.Column, "string has no property '%s'", propertyName)
		}
	case *object.Hash:
		propertyName := node.Property.Value

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
		return newError(node.Token.Line, node.Token.Column, "dot operator not supported for %s", obj.Type())
	}
}

func evalMethodCallExpression(obj object.Object, method string, args []object.Object, line, column int) object.Object {
	switch obj := obj.(type) {
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

			length := len(obj.Elements)
			newElements := make([]object.Object, length+1)
			copy(newElements, obj.Elements)
			newElements[length] = args[0]

			return &object.Array{Elements: newElements}
		case "rest":
			if len(args) != 0 {
				return newError(line, column, "wrong number of arguments for array.rest(): got %d, want 0", len(args))
			}

			length := len(obj.Elements)
			if length > 0 {
				newElements := make([]object.Object, length-1)
				copy(newElements, obj.Elements[1:])
				return &object.Array{Elements: newElements}
			}
			return NULL
		default:
			return newError(line, column, "array has no method '%s'", method)
		}
	case *object.Hash:
		switch method {
		case "keys":
			if len(args) != 0 {
				return newError(line, column, "wrong number of arguments for object.keys(): got %d, want 0", len(args))
			}

			keys := []object.Object{}
			for _, pair := range obj.Pairs {
				keys = append(keys, pair.Key)
			}

			return &object.Array{Elements: keys}

		case "values":
			if len(args) != 0 {
				return newError(line, column, "wrong number of arguments for object.values(): got %d, want 0", len(args))
			}

			values := []object.Object{}
			for _, pair := range obj.Pairs {
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
			return newError(line, column, "object has no method '%s'", method)
		}
	default:
		return newError(line, column, "method call not supported for %s", obj.Type())
	}
}

func evalRepeatStatement(rs *ast.RepeatStatement, env *object.Environment) object.Object {
	loopEnv := object.NewEnclosedEnvironment(env)

	if rs.Iterator != nil && rs.Collection != nil {
		collection := Eval(rs.Collection, env)
		if isError(collection) {
			return collection
		}

		switch collection := collection.(type) {
		case *object.Array:
			for _, element := range collection.Elements {
				loopEnv.Set(rs.Iterator.Value, element)

				result := Eval(rs.Body, loopEnv)

				if result != nil {
					if result.Type() == object.RETURN_VALUE_OBJ || result.Type() == object.ERROR_OBJ {
						return result
					}

					if result.Type() == object.BREAK_OBJ {
						break
					}

					if result.Type() == object.CONTINUE_OBJ {
						continue
					}
				}
			}

		case *object.Integer:
			end := collection.Value

			for i := int64(1); i <= end; i++ {
				loopEnv.Set(rs.Iterator.Value, &object.Integer{Value: i})

				result := Eval(rs.Body, loopEnv)
				if result != nil {
					if result.Type() == object.RETURN_VALUE_OBJ || result.Type() == object.ERROR_OBJ {
						return result
					}

					if result.Type() == object.BREAK_OBJ {
						break
					}

					if result.Type() == object.CONTINUE_OBJ {
						continue
					}
				}
			}

		default:
			return newError(rs.Token.Line, rs.Token.Column,
				"cannot iterate over %s", collection.Type())
		}

		return NULL
	}

	if rs.Collection != nil && rs.Iterator == nil {
		for {
			condition := Eval(rs.Collection, env)
			if isError(condition) {
				return condition
			}

			if !isTruthy(condition) {
				break
			}

			result := Eval(rs.Body, env)
			if result != nil {
				if result.Type() == object.RETURN_VALUE_OBJ || result.Type() == object.ERROR_OBJ {
					return result
				}

				if result.Type() == object.BREAK_OBJ {
					break
				}

				if result.Type() == object.CONTINUE_OBJ {
					continue
				}
			}
		}

		return NULL
	}

	return newError(rs.Token.Line, rs.Token.Column,
		"invalid repeat statement: missing iterator or condition")
}

func evalPrintExpression(pe *ast.PrintStatement, env *object.Environment) object.Object {
	args := []object.Object{}

	for _, arg := range pe.Arguments {
		evaluated := Eval(arg, env)
		if isError(evaluated) {
			return evaluated
		}
		args = append(args, evaluated)
	}

	outputs := make([]string, len(args))
	for i, arg := range args {
		outputs[i] = arg.Inspect()
	}

	if len(outputs) > 0 {
		fmt.Fprintln(os.Stdout, strings.Join(outputs, " "))
	} else {
		fmt.Fprintln(os.Stdout)
	}

	return NULL
}
