//go:build tinygo && wasm

// Command wasm is the TargaScript interpreter for web pages and workers,
// loaded by web/targascript.js. Scripts see JavaScript's globalThis as `js`.
package main

import (
	"strings"

	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/jsbridge"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
)

// maxNesting keeps recursion within the browser's native stack, allowing
// about 400 nested calls of a simple recursive function. Measured in Node,
// whose stack is the size of Chrome's, failures started near 1850.
const maxNesting = 1200

// Environment shared by the scripts of a page
var pageEnv *object.Environment

// writer sends output to a function, a TargaScript one or a JavaScript one.
type writer func(text string)

func (w writer) Write(p []byte) (int, error) {
	w(string(p))
	return len(p), nil
}

func newEnvironment() *object.Environment {
	env := object.NewEnvironment()
	env.Set("NULL", evaluator.NULL)
	env.Set("js", jsbridge.Global())
	env.Set("targascript", targascriptObject())
	return env
}

// current output, restored after a nested run
var output writer

// run parses and evaluates source in env, returning the error message or
// "".
func run(source string, env *object.Environment, out writer) string {
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return strings.Join(p.Errors(), "\n")
	}

	previous := output
	output = out
	evaluator.SetOutput(out)
	defer func() {
		output = previous
		evaluator.SetOutput(previous)
	}()

	if err, ok := evaluator.Eval(program, env).(*object.Error); ok {
		return jsbridge.ErrorMessage(err)
	}
	return ""
}

// targascriptObject is the `targascript` value scripts see. Its run function
// runs another program in a fresh environment, as the playground does in a
// worker, and returns the error message or null.
func targascriptObject() *object.Hash {
	hash := &object.Hash{Pairs: map[object.HashKey]object.HashPair{}}
	hash.Set(&object.String{Value: "run"}, &object.Builtin{
		Name: "run",
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Message: "wrong number of arguments for targascript.run. got=" +
					(&object.Integer{Value: int64(len(args))}).Inspect() + ", want=2"}
			}
			source, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: "first argument to targascript.run must be a STRING"}
			}
			onOutput := args[1]
			if !evaluator.IsCallable(onOutput) {
				return &object.Error{Message: "second argument to targascript.run must be a function"}
			}
			out := writer(func(text string) {
				evaluator.CallFunction(onOutput, []object.Object{&object.String{Value: text}})
			})
			if err := run(source.Value, newEnvironment(), out); err != "" {
				return &object.String{Value: err}
			}
			return evaluator.NULL
		},
	})
	return hash
}

// evaluate runs the string with handle source, sending output to the
// JavaScript function onOutput. With fresh set it uses a new environment,
// otherwise the page's. It returns the handle of the error message, or 0
// (undefined).
//
//go:wasmexport evaluate
func evaluate(source, onOutput, fresh uint32) uint32 {
	env := pageEnv
	if fresh != 0 {
		env = newEnvironment()
	}
	out := writer(func(text string) { jsbridge.Invoke(onOutput, text) })
	if err := run(jsbridge.ReadString(source), env, out); err != "" {
		return jsbridge.MakeString(err)
	}
	return 0
}

// callback runs a TargaScript function passed to JavaScript.
//
//go:wasmexport callback
func callback(id, args uint32) uint32 {
	return jsbridge.Callback(id, args)
}

func init() {
	evaluator.SetMaxNesting(maxNesting)
	pageEnv = newEnvironment()
}

func main() {}
