package evaluator

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/migueltarga/TargaScript/object"
)

var builtins map[string]*object.Builtin

// initBuiltins initializes all built-in functions
func init() {
	builtins = map[string]*object.Builtin{
		"print": {
			Name: "print",
			Fn: func(args ...object.Object) object.Object {
				for _, arg := range args {
					fmt.Println(arg.Inspect())
				}

				return NULL
			},
		},

		// printf - Formatted printing with support for format specifiers
		"printf": {
			Name: "printf",
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 {
					return &object.Error{
						Message: "wrong number of arguments for printf. got=0, want at least 1",
					}
				}

				format, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{
						Message: fmt.Sprintf("first argument to printf must be a STRING, got %s", args[0].Type()),
					}
				}

				values := make([]interface{}, len(args)-1)
				for i, arg := range args[1:] {
					switch arg := arg.(type) {
					case *object.Integer:
						values[i] = arg.Value
					case *object.Float:
						values[i] = arg.Value
					case *object.Boolean:
						values[i] = arg.Value
					case *object.String:
						values[i] = arg.Value
					case *object.Null:
						values[i] = "null"
					default:
						values[i] = arg.Inspect()
					}
				}

				// Use the format with the values
				fmt.Fprintf(output, format.Value, values...)
				fmt.Fprintln(output) // add newline at the end

				return NULL
			},
		},

		// Add a built-in range function for creating arrays of sequential integers
		"range": {
			Name: "range",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for range. got=%d, want=2", len(args)),
					}
				}

				start, ok := args[0].(*object.Integer)
				if !ok {
					return &object.Error{
						Message: fmt.Sprintf("first argument to range must be INTEGER, got %s", args[0].Type()),
					}
				}

				end, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{
						Message: fmt.Sprintf("second argument to range must be INTEGER, got %s", args[1].Type()),
					}
				}

				startVal := start.Value
				endVal := end.Value

				// If start > end, return empty array
				if startVal > endVal {
					return &object.Array{Elements: []object.Object{}}
				}

				// Create the range array
				elements := make([]object.Object, 0, endVal-startVal+1)
				for i := startVal; i <= endVal; i++ {
					elements = append(elements, &object.Integer{Value: i})
				}

				// Log built-in range function call for debugging
				if traceMode {
					fmt.Printf("TRACE: Built-in range(%d, %d) created array with %d elements\n",
						startVal, endVal, len(elements))
				}

				return &object.Array{Elements: elements}
			},
		},

		// type - Return the type of an object as a string
		"type": {
			Name: "type",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for type. got=%d, want=1", len(args)),
					}
				}

				return &object.String{Value: string(args[0].Type())}
			},
		},

		// toString - Convert a value to a string
		"toString": {
			Name: "toString",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for toString. got=%d, want=1", len(args)),
					}
				}

				switch arg := args[0].(type) {
				case *object.String:
					return arg
				case *object.Integer:
					return &object.String{Value: fmt.Sprintf("%d", arg.Value)}
				case *object.Float:
					return &object.String{Value: fmt.Sprintf("%g", arg.Value)}
				case *object.Boolean:
					return &object.String{Value: fmt.Sprintf("%t", arg.Value)}
				case *object.Null:
					return &object.String{Value: "null"}
				default:
					return &object.String{Value: arg.Inspect()}
				}
			},
		},

		// toInt - Convert a value to an integer
		"toInt": {
			Name: "toInt",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for toInt. got=%d, want=1", len(args)),
					}
				}

				switch arg := args[0].(type) {
				case *object.Integer:
					return arg
				case *object.Float:
					return &object.Integer{Value: int64(arg.Value)}
				case *object.String:
					value, err := strconv.ParseInt(arg.Value, 10, 64)
					if err != nil {
						return &object.Error{
							Message: fmt.Sprintf("could not convert string '%s' to INTEGER", arg.Value),
						}
					}
					return &object.Integer{Value: value}
				case *object.Boolean:
					if arg.Value {
						return &object.Integer{Value: 1}
					}
					return &object.Integer{Value: 0}
				default:
					return &object.Error{
						Message: fmt.Sprintf("argument to `toInt` not supported, got %s", args[0].Type()),
					}
				}
			},
		},

		// toFloat - Convert a value to a float
		"toFloat": {
			Name: "toFloat",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for toFloat. got=%d, want=1", len(args)),
					}
				}

				switch arg := args[0].(type) {
				case *object.Float:
					return arg
				case *object.Integer:
					return &object.Float{Value: float64(arg.Value)}
				case *object.String:
					value, err := strconv.ParseFloat(arg.Value, 64)
					if err != nil {
						return &object.Error{
							Message: fmt.Sprintf("could not convert string '%s' to FLOAT", arg.Value),
						}
					}
					return &object.Float{Value: value}
				case *object.Boolean:
					if arg.Value {
						return &object.Float{Value: 1.0}
					}
					return &object.Float{Value: 0.0}
				default:
					return &object.Error{
						Message: fmt.Sprintf("argument to `toFloat` not supported, got %s", args[0].Type()),
					}
				}
			},
		},

		// Math functions
		"abs": {
			Name: "abs",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for abs. got=%d, want=1", len(args)),
					}
				}

				switch arg := args[0].(type) {
				case *object.Integer:
					if arg.Value < 0 {
						return &object.Integer{Value: -arg.Value}
					}
					return arg
				case *object.Float:
					return &object.Float{Value: math.Abs(arg.Value)}
				default:
					return &object.Error{
						Message: fmt.Sprintf("argument to `abs` must be a number, got %s", args[0].Type()),
					}
				}
			},
		},

		// sqrt - Square root of a number
		"sqrt": {
			Name: "sqrt",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for sqrt. got=%d, want=1", len(args)),
					}
				}

				var value float64
				switch arg := args[0].(type) {
				case *object.Integer:
					value = float64(arg.Value)
				case *object.Float:
					value = arg.Value
				default:
					return &object.Error{
						Message: fmt.Sprintf("argument to `sqrt` must be a number, got %s", args[0].Type()),
					}
				}

				if value < 0 {
					return &object.Error{
						Message: "cannot take square root of negative number",
					}
				}

				return &object.Float{Value: math.Sqrt(value)}
			},
		},

		// pow - Power function
		"pow": {
			Name: "pow",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for pow. got=%d, want=2", len(args)),
					}
				}

				var base, exponent float64

				switch arg := args[0].(type) {
				case *object.Integer:
					base = float64(arg.Value)
				case *object.Float:
					base = arg.Value
				default:
					return &object.Error{
						Message: fmt.Sprintf("first argument to `pow` must be a number, got %s", args[0].Type()),
					}
				}

				switch arg := args[1].(type) {
				case *object.Integer:
					exponent = float64(arg.Value)
				case *object.Float:
					exponent = arg.Value
				default:
					return &object.Error{
						Message: fmt.Sprintf("second argument to `pow` must be a number, got %s", args[1].Type()),
					}
				}

				return &object.Float{Value: math.Pow(base, exponent)}
			},
		},

		// round - Round a number to the nearest integer
		"round": {
			Name: "round",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for round. got=%d, want=1", len(args)),
					}
				}

				var value float64
				switch arg := args[0].(type) {
				case *object.Integer:
					return arg
				case *object.Float:
					value = arg.Value
				default:
					return &object.Error{
						Message: fmt.Sprintf("argument to `round` must be a number, got %s", args[0].Type()),
					}
				}

				return &object.Integer{Value: int64(math.Round(value))}
			},
		},

		// floor - Round a number down to the nearest integer
		"floor": {
			Name: "floor",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for floor. got=%d, want=1", len(args)),
					}
				}

				var value float64
				switch arg := args[0].(type) {
				case *object.Integer:
					return arg
				case *object.Float:
					value = arg.Value
				default:
					return &object.Error{
						Message: fmt.Sprintf("argument to `floor` must be a number, got %s", args[0].Type()),
					}
				}

				return &object.Integer{Value: int64(math.Floor(value))}
			},
		},

		// ceil - Round a number up to the nearest integer
		"ceil": {
			Name: "ceil",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for ceil. got=%d, want=1", len(args)),
					}
				}

				var value float64
				switch arg := args[0].(type) {
				case *object.Integer:
					return arg
				case *object.Float:
					value = arg.Value
				default:
					return &object.Error{
						Message: fmt.Sprintf("argument to `ceil` must be a number, got %s", args[0].Type()),
					}
				}

				return &object.Integer{Value: int64(math.Ceil(value))}
			},
		},

		// min - Minimum of two numbers
		"min": {
			Name: "min",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for min. got=%d, want=2", len(args)),
					}
				}

				var val1, val2 float64

				switch arg := args[0].(type) {
				case *object.Integer:
					val1 = float64(arg.Value)
				case *object.Float:
					val1 = arg.Value
				default:
					return &object.Error{
						Message: fmt.Sprintf("first argument to `min` must be a number, got %s", args[0].Type()),
					}
				}

				switch arg := args[1].(type) {
				case *object.Integer:
					val2 = float64(arg.Value)
				case *object.Float:
					val2 = arg.Value
				default:
					return &object.Error{
						Message: fmt.Sprintf("second argument to `min` must be a number, got %s", args[1].Type()),
					}
				}

				if val1 <= val2 {
					return args[0]
				}
				return args[1]
			},
		},

		// max - Maximum of two numbers
		"max": {
			Name: "max",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for max. got=%d, want=2", len(args)),
					}
				}

				var val1, val2 float64

				switch arg := args[0].(type) {
				case *object.Integer:
					val1 = float64(arg.Value)
				case *object.Float:
					val1 = arg.Value
				default:
					return &object.Error{
						Message: fmt.Sprintf("first argument to `max` must be a number, got %s", args[0].Type()),
					}
				}

				switch arg := args[1].(type) {
				case *object.Integer:
					val2 = float64(arg.Value)
				case *object.Float:
					val2 = arg.Value
				default:
					return &object.Error{
						Message: fmt.Sprintf("second argument to `max` must be a number, got %s", args[1].Type()),
					}
				}

				if val1 >= val2 {
					return args[0]
				}
				return args[1]
			},
		},

		// random - Generate a random number between 0 and 1
		"random": {
			Name: "random",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 0 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for random. got=%d, want=0", len(args)),
					}
				}

				// Using current time nanoseconds to generate a pseudo-random number between 0 and 1
				// This is not cryptographically secure but sufficient for basic randomness
				return &object.Float{Value: float64(time.Now().UnixNano()%10000) / 10000.0}
			},
		},

		// Time functions
		"now": {
			Name: "now",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 0 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for now. got=%d, want=0", len(args)),
					}
				}

				return &object.Integer{Value: time.Now().Unix()}
			},
		},

		// timeFormat - Format a timestamp
		"timeFormat": {
			Name: "timeFormat",
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{
						Message: fmt.Sprintf("wrong number of arguments for timeFormat. got=%d, want=2", len(args)),
					}
				}

				timestamp, ok := args[0].(*object.Integer)
				if !ok {
					return &object.Error{
						Message: fmt.Sprintf("first argument to `timeFormat` must be INTEGER, got %s", args[0].Type()),
					}
				}

				format, ok := args[1].(*object.String)
				if !ok {
					return &object.Error{
						Message: fmt.Sprintf("second argument to `timeFormat` must be STRING, got %s", args[1].Type()),
					}
				}

				t := time.Unix(timestamp.Value, 0)
				formatted := t.Format(format.Value)

				return &object.String{Value: formatted}
			},
		},
	}

}
