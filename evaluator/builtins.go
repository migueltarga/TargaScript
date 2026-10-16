package evaluator

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/migueltarga/TargaScript/object"
)

// builtins are the builtin functions, by name. Files other than this one
// add theirs in their own init functions, which can run before this one.
var builtins = map[string]*object.Builtin{}

// Host runs the asynchronous work of builtins like sleep: an event loop in
// the CLI, JavaScript's timers in the browser.
type Host interface {
	// After calls fn after the given number of milliseconds, on the
	// goroutine that runs the program.
	After(milliseconds int64, fn func())

	// Fetch makes an HTTP request and calls done with the response, or
	// with the error that prevented it, on the goroutine that runs the
	// program.
	Fetch(request FetchRequest, done func(FetchResponse, error))
}

// FetchRequest is an HTTP request made by fetch.
type FetchRequest struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    string
}

// FetchResponse is the response to a FetchRequest, with its whole body.
type FetchResponse struct {
	URL     string
	Status  int
	Headers map[string]string
	Body    string
}

var host Host

// SetHost sets where asynchronous builtins schedule their work. Without a
// host, they fail.
func SetHost(h Host) {
	host = h
}

// initBuiltins initializes all built-in functions
func init() {
	for name, builtin := range map[string]*object.Builtin{
		// print - Print values separated by spaces, followed by a newline
		"print": {
			Name: "print",
			Fn: func(args ...object.Object) object.Object {
				printValues(args)
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
						Message: fmt.Sprintf("first argument to printf must be a string, got %s", typeName(args[0])),
					}
				}

				text, err := formatPrintf(format.Value, args[1:])
				if err != nil {
					return err
				}
				// A line break ends the output, unless the format has its own
				if !strings.HasSuffix(text, "\n") {
					text += "\n"
				}
				fmt.Fprint(output, text)

				return NULL
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

				return &object.String{Value: kindName(args[0])}
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
					return &object.String{Value: arg.Inspect()}
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
					value, ok := floatToInt(arg.Value)
					if !ok {
						return &object.Error{Message: "integer overflow"}
					}
					return &object.Integer{Value: value}
				case *object.String:
					value, err := strconv.ParseInt(arg.Value, 10, 64)
					if err != nil {
						return &object.Error{
							Message: fmt.Sprintf("cannot convert %q to an integer", arg.Value),
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
						Message: fmt.Sprintf("cannot convert %s to an integer", typeName(args[0])),
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
							Message: fmt.Sprintf("cannot convert %q to a float", arg.Value),
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
						Message: fmt.Sprintf("cannot convert %s to a float", typeName(args[0])),
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
						value, ok := subInt(0, arg.Value)
						if !ok {
							return &object.Error{Message: "integer overflow"}
						}
						return &object.Integer{Value: value}
					}
					return arg
				case *object.Float:
					return &object.Float{Value: math.Abs(arg.Value)}
				default:
					return &object.Error{
						Message: fmt.Sprintf("argument to `abs` must be a number, got %s", typeName(args[0])),
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
						Message: fmt.Sprintf("argument to `sqrt` must be a number, got %s", typeName(args[0])),
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
						Message: fmt.Sprintf("first argument to `pow` must be a number, got %s", typeName(args[0])),
					}
				}

				switch arg := args[1].(type) {
				case *object.Integer:
					exponent = float64(arg.Value)
				case *object.Float:
					exponent = arg.Value
				default:
					return &object.Error{
						Message: fmt.Sprintf("second argument to `pow` must be a number, got %s", typeName(args[1])),
					}
				}

				return &object.Float{Value: math.Pow(base, exponent)}
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
						Message: fmt.Sprintf("argument to `floor` must be a number, got %s", typeName(args[0])),
					}
				}

				result, ok := floatToInt(math.Floor(value))
				if !ok {
					return &object.Error{Message: "integer overflow"}
				}
				return &object.Integer{Value: result}
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
						Message: fmt.Sprintf("argument to `ceil` must be a number, got %s", typeName(args[0])),
					}
				}

				result, ok := floatToInt(math.Ceil(value))
				if !ok {
					return &object.Error{Message: "integer overflow"}
				}
				return &object.Integer{Value: result}
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
						Message: fmt.Sprintf("first argument to `timeFormat` must be an integer, got %s", typeName(args[0])),
					}
				}

				format, ok := args[1].(*object.String)
				if !ok {
					return &object.Error{
						Message: fmt.Sprintf("second argument to `timeFormat` must be a string, got %s", typeName(args[1])),
					}
				}

				t := time.Unix(timestamp.Value, 0)
				formatted := t.Format(dateLayout(format.Value))

				return &object.String{Value: formatted}
			},
		},
	} {
		builtins[name] = builtin
	}
}
