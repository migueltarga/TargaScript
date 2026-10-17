// Package targascript runs TargaScript from Go programs, as an embedded
// scripting language: for rules, plugins, formulas or configuration that
// users write.
//
//	script := targascript.New()
//	script.Set("price", 120.0)
//	script.Set("discount", func(p float64) float64 { return p * 0.9 })
//	total, err := script.Run(`discount(price) |> (p => round(p, 2))`)
//
// Values cross between Go and TargaScript like JSON: numbers, strings,
// booleans, nil, slices and maps, plus Go functions, which scripts can call,
// and TargaScript functions, which Go can call. Scripts can't reach files,
// the environment or the exit code; they can use fetch and sleep.
package targascript

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/migueltarga/TargaScript/compiler"
	"github.com/migueltarga/TargaScript/diag"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/eventloop"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
	"github.com/migueltarga/TargaScript/vm"
)

// Script is a TargaScript environment: what one Run defines, the next sees.
// It isn't safe to use from several goroutines at once.
type Script struct {
	compiler *compiler.Compiler
	machine  *vm.VM
	loop     *eventloop.Loop
}

// New returns an empty environment.
func New() *Script {
	c := compiler.New()
	c.AllowLoad = false
	l := eventloop.NewRestricted()
	machine := vm.New(c.Globals())
	machine.SetHost(l)
	machine.RestrictBuiltins("readFile", "readBytes", "writeFile", "appendFile", "listDir", "env", "exit", "input", "exec", "exists", "deleteFile", "makeDir", "staticFile", "serve")
	return &Script{compiler: c, machine: machine, loop: l}
}

// SetOutput sets where print writes; standard output when it isn't set.
func (s *Script) SetOutput(w io.Writer) {
	s.machine.Output = w
}

// SetMaxSteps stops a Run with an error after that many loop iterations and
// calls, to bound scripts that might not end. 0 means no limit.
func (s *Script) SetMaxSteps(n int) {
	s.machine.MaxSteps = n
}

// AllowFetchHosts enables fetch for the given exact hostnames. Fetch is
// disabled by default in embedded scripts.
func (s *Script) AllowFetchHosts(hosts ...string) {
	s.loop.AllowFetchHosts(hosts...)
}

// SetMaxFetchBytes sets the largest response body an embedded script can
// receive. The default is 1 MiB.
func (s *Script) SetMaxFetchBytes(n int64) {
	s.loop.SetMaxFetchBytes(n)
}

// Error is an error in a script: its message and where it happened.
type Error struct {
	Message      string
	Line, Column int
	source       string
}

// Error describes the error with the line of code it points at.
func (e *Error) Error() string {
	return diag.Format(e.source, diag.Diagnostic{Line: e.Line, Column: e.Column, Message: e.Message})
}

// Run runs source and returns the value of its last expression, waiting
// for it when the script awaits.
func (s *Script) Run(source string) (any, error) {
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		d := p.Diagnostics()[0]
		return nil, &Error{Message: d.Message, Line: d.Line, Column: d.Column, source: source}
	}
	bytecode, err := s.compiler.Compile(program)
	if err != nil {
		return nil, &Error{Message: err.Error(), source: source}
	}
	result, err := wait(s.loop, s.machine.Run(bytecode))
	if err != nil {
		if scriptErr, ok := err.(*Error); ok {
			scriptErr.source = source
		}
		return nil, err
	}
	return FromValue(result), nil
}

// wait runs the event loop until a script that paused at an await is done.
func wait(loop *eventloop.Loop, result object.Object) (object.Object, error) {
	if promise, ok := result.(*object.Promise); ok {
		if loop == nil || !loop.Run(promise.Settled) {
			return nil, errors.New("the script is waiting for something that will never happen")
		}
		value, rejected := promise.Result()
		if rejected {
			if err, ok := value.(*object.Error); ok {
				return nil, &Error{Message: err.Message, Line: err.Line, Column: err.Column}
			}
			return nil, &Error{Message: evaluator.ToDisplayString(value)}
		}
		result = value
	}
	if err, ok := result.(*object.Error); ok {
		return nil, &Error{Message: err.Message, Line: err.Line, Column: err.Column}
	}
	return result, nil
}

// Set gives the script a global variable with a Go value; see ToValue.
func (s *Script) Set(name string, value any) error {
	converted, err := ToValue(value)
	if err != nil {
		return err
	}
	s.compiler.Globals().Define(name)
	s.machine.SetGlobal(name, converted)
	return nil
}

// Get returns the value of a global variable as a Go value, or nil.
func (s *Script) Get(name string) any {
	value := s.machine.Global(name)
	if value == nil {
		return nil
	}
	return FromValue(value)
}

// Call calls a function the script defined, with Go values.
func (s *Script) Call(name string, args ...any) (any, error) {
	fn := s.machine.Global(name)
	if fn == nil || !evaluator.IsCallable(fn) {
		return nil, fmt.Errorf("%s is not a function", name)
	}
	return callValue(fn, args)
}

func callValue(fn object.Object, args []any) (any, error) {
	values := make([]object.Object, len(args))
	for i, arg := range args {
		value, err := ToValue(arg)
		if err != nil {
			return nil, err
		}
		values[i] = value
	}
	var loop *eventloop.Loop
	if closure, ok := fn.(*object.Closure); ok {
		if machine, ok := closure.Runner.(*vm.VM); ok {
			loop, _ = machine.Host().(*eventloop.Loop)
		}
	}
	result, err := wait(loop, evaluator.CallFunction(fn, values))
	if err != nil {
		return nil, err
	}
	return FromValue(result), nil
}

// ToValue converts a Go value for a script:
//
//   - nil, bools, numbers and strings as they are;
//   - slices and arrays as arrays, maps as objects, with their keys sorted;
//   - structs as objects of their exported fields, named by their json tags;
//   - functions as functions the script can call. Their arguments are
//     converted with FromValue, and a last result of type error that isn't
//     nil fails the call in the script.
func ToValue(value any) (object.Object, error) {
	if value == nil {
		return evaluator.NULL, nil
	}
	if obj, ok := value.(object.Object); ok {
		return obj, nil
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			return evaluator.TRUE, nil
		}
		return evaluator.FALSE, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &object.Integer{Value: v.Int()}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &object.Integer{Value: int64(v.Uint())}, nil
	case reflect.Float32, reflect.Float64:
		return &object.Float{Value: v.Float()}, nil
	case reflect.String:
		return &object.String{Value: v.String()}, nil
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return evaluator.NULL, nil
		}
		elements := make([]object.Object, v.Len())
		for i := range elements {
			element, err := ToValue(v.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			elements[i] = element
		}
		return &object.Array{Elements: elements}, nil
	case reflect.Map:
		if v.IsNil() {
			return evaluator.NULL, nil
		}
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j]) })
		hash := object.NewHash()
		for _, key := range keys {
			element, err := ToValue(v.MapIndex(key).Interface())
			if err != nil {
				return nil, err
			}
			hash.Set(&object.String{Value: fmt.Sprint(key.Interface())}, element)
		}
		return hash, nil
	case reflect.Struct:
		hash := object.NewHash()
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			name, skip := fieldName(field)
			if skip {
				continue
			}
			element, err := ToValue(v.Field(i).Interface())
			if err != nil {
				return nil, err
			}
			hash.Set(&object.String{Value: name}, element)
		}
		return hash, nil
	case reflect.Func:
		return goFunction(v), nil
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return evaluator.NULL, nil
		}
		return ToValue(v.Elem().Interface())
	}
	return nil, fmt.Errorf("cannot give a %T to a script", value)
}

// fieldName is the property a struct field becomes: its json tag's name
// when it has one, else its name. Unexported fields and json:"-" are
// skipped.
func fieldName(field reflect.StructField) (string, bool) {
	if !field.IsExported() {
		return "", true
	}
	tag := field.Tag.Get("json")
	name, _, _ := strings.Cut(tag, ",")
	switch name {
	case "-":
		return "", true
	case "":
		return field.Name, false
	}
	return name, false
}

// goFunction wraps a Go function so scripts can call it.
func goFunction(fn reflect.Value) *object.Builtin {
	t := fn.Type()
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	return &object.Builtin{Name: t.String(), Fn: func(args ...object.Object) object.Object {
		in := make([]reflect.Value, 0, len(args))
		for i, arg := range args {
			var param reflect.Type
			switch {
			case t.IsVariadic() && i >= t.NumIn()-1:
				param = t.In(t.NumIn() - 1).Elem()
			case i < t.NumIn():
				param = t.In(i)
			default:
				return &object.Error{Message: fmt.Sprintf("too many arguments: the function takes %d", t.NumIn())}
			}
			value, err := convertArgument(FromValue(arg), param)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("argument %d: %s", i+1, err)}
			}
			in = append(in, value)
		}
		minimum := t.NumIn()
		if t.IsVariadic() {
			minimum--
		}
		for len(in) < minimum {
			in = append(in, reflect.Zero(t.In(len(in))))
		}

		out := fn.Call(in)
		if n := len(out); n > 0 && t.Out(n-1) == errorType {
			if err, _ := out[n-1].Interface().(error); err != nil {
				return &object.Error{Message: err.Error()}
			}
			out = out[:n-1]
		}
		if len(out) == 0 {
			return evaluator.NULL
		}
		result, err := ToValue(out[0].Interface())
		if err != nil {
			return &object.Error{Message: err.Error()}
		}
		return result
	}}
}

// convertArgument turns a value from FromValue into a Go function's
// parameter type.
func convertArgument(value any, param reflect.Type) (reflect.Value, error) {
	if value == nil {
		return reflect.Zero(param), nil
	}
	v := reflect.ValueOf(value)
	switch {
	case v.Type().AssignableTo(param):
		return v, nil
	case v.Type().ConvertibleTo(param) && isNumber(v.Kind()) && isNumber(param.Kind()):
		return v.Convert(param), nil
	}
	return reflect.Value{}, fmt.Errorf("cannot use %T as %s", value, param)
}

func isNumber(k reflect.Kind) bool {
	return k >= reflect.Int && k <= reflect.Float64
}

// FromValue converts a script's value for Go: integers as int64, floats as
// float64, strings, bools, nil, arrays as []any, objects as map[string]any,
// and functions as func(args ...any) (any, error). A value inside itself
// becomes nil there.
func FromValue(value object.Object) any {
	return fromValue(value, nil)
}

func fromValue(value object.Object, parents []object.Object) any {
	switch value.(type) {
	case *object.Array, *object.Hash:
		for _, parent := range parents {
			if parent == value {
				return nil
			}
		}
		parents = append(parents, value)
	}

	switch value := value.(type) {
	case *object.Integer:
		return value.Value
	case *object.Float:
		return value.Value
	case *object.String:
		return value.Value
	case *object.Boolean:
		return value.Value
	case *object.Null:
		return nil
	case *object.Array:
		elements := make([]any, len(value.Elements))
		for i, element := range value.Elements {
			elements[i] = fromValue(element, parents)
		}
		return elements
	case *object.Hash:
		m := make(map[string]any, value.Len())
		for _, pair := range value.OrderedPairs() {
			m[evaluator.ToDisplayString(pair.Key)] = fromValue(pair.Value, parents)
		}
		return m
	}
	if evaluator.IsCallable(value) {
		return func(args ...any) (any, error) { return callValue(value, args) }
	}
	return value
}
