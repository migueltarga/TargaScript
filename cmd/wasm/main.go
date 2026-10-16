//go:build tinygo && wasm

// Command wasm is the TargaScript interpreter for web pages and workers,
// loaded by web/targascript.js. Scripts see JavaScript's globalThis as `js`.
package main

import (
	"strings"

	"github.com/migueltarga/TargaScript/compiler"
	"github.com/migueltarga/TargaScript/diag"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/jsbridge"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
	"github.com/migueltarga/TargaScript/vm"
)

// maxNestedRuns limits closures called from Go code, like the callbacks of
// map or JavaScript events, which nest in the browser's small native stack.
// Ordinary recursion runs in the VM's own frames and isn't limited by it.
const maxNestedRuns = 100

// Session shared by the scripts of a page
var page *session

// writer sends output to a function, a TargaScript one or a JavaScript one.
type writer func(text string)

func (w writer) Write(p []byte) (int, error) {
	w(string(p))
	return len(p), nil
}

// session compiles and runs code, keeping its variables between runs.
type session struct {
	compiler *compiler.Compiler
	machine  *vm.VM
}

func newSession() *session {
	c := compiler.New()
	s := &session{compiler: c, machine: vm.New(c.Globals())}
	s.machine.SetGlobal("NULL", evaluator.NULL)
	s.machine.SetGlobal("js", jsbridge.Global())
	s.machine.SetGlobal("targascript", targascriptObject())
	return s
}

// current output, restored after a nested run
var output writer = toConsole

// toConsole is where print writes outside a run, as in event handlers. Each
// print is one write ending in a newline, which console.log adds itself.
func toConsole(text string) {
	jsbridge.ConsoleLog(strings.TrimSuffix(text, "\n"))
}

// run parses and evaluates source in env, returning the error message or
// "".
func run(source string, s *session, out writer) string {
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		diagnostics := []diag.Diagnostic{}
		for _, e := range p.Diagnostics() {
			diagnostics = append(diagnostics, diag.Diagnostic{Line: e.Line, Column: e.Column, Message: e.Message})
		}
		return diag.FormatAll(source, diagnostics)
	}

	previous := output
	output = out
	evaluator.SetOutput(out)
	defer func() {
		output = previous
		evaluator.SetOutput(previous)
	}()

	bytecode, err := s.compiler.Compile(program)
	if err != nil {
		return err.Error()
	}
	if err, ok := s.machine.Run(bytecode).(*object.Error); ok {
		return diag.Format(source, diag.Diagnostic{Line: err.Line, Column: err.Column, Message: err.Message})
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
			if err := run(source.Value, newSession(), out); err != "" {
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
	s := page
	if fresh != 0 {
		s = newSession()
	}
	out := writer(func(text string) { jsbridge.Invoke(onOutput, text) })
	if err := run(jsbridge.ReadString(source), s, out); err != "" {
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
	vm.MaxNestedRuns = maxNestedRuns
	evaluator.SetOutput(output)
	page = newSession()
}

func main() {}
