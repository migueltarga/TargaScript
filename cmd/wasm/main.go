//go:build tinygo && wasm

// Command wasm is the TargaScript interpreter for web pages and workers,
// loaded by web/targascript.js. Scripts see JavaScript's globalThis as `js`.
package main

import (
	"strings"

	"github.com/migueltarga/TargaScript/compiler"
	"github.com/migueltarga/TargaScript/diag"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/format"
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

// toConsole is where print writes outside a run, as in event handlers. Each
// print is one write ending in a newline, which console.log adds itself.
func toConsole(text string) {
	jsbridge.ConsoleLog(strings.TrimSuffix(text, "\n"))
}

// run parses and runs source in a session, sending its output to out. It
// returns null when the program finished, the error message as a string
// when it failed, or a promise of one of those when it paused at an await.
func run(source string, s *session, out writer) object.Object {
	result, failure := execute(source, s, out)
	if failure != "" {
		return &object.String{Value: failure}
	}
	return outcome(source, result)
}

// evalValue runs source in a session like run, but returns the value of the
// code, or an error with the message run would return.
func evalValue(source string, s *session, out writer) object.Object {
	result, failure := execute(source, s, out)
	if failure != "" {
		return &object.Error{Message: failure}
	}
	if e, ok := result.(*object.Error); ok {
		return &object.Error{Message: diag.Format(source, evaluator.Diagnostic(e))}
	}
	return result
}

// execute parses, compiles and runs source in a session, returning the
// result of the run, or the message of the errors that kept it from
// running.
func execute(source string, s *session, out writer) (object.Object, string) {
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		diagnostics := []diag.Diagnostic{}
		for _, e := range p.Diagnostics() {
			diagnostics = append(diagnostics, diag.Diagnostic{Line: e.Line, Column: e.Column, Message: e.Message})
		}
		return nil, diag.FormatAll(source, diagnostics)
	}
	bytecode, err := s.compiler.Compile(program)
	if err != nil {
		return nil, err.Error()
	}
	s.machine.Output = out
	return s.machine.Run(bytecode), ""
}

// outcome turns the result of a run into what run returns.
func outcome(source string, result object.Object) object.Object {
	switch result := result.(type) {
	case *object.Error:
		return &object.String{Value: diag.Format(source, evaluator.Diagnostic(result))}
	case *object.Promise:
		done := object.NewPromise()
		result.Then(
			func(object.Object) { done.Resolve(evaluator.NULL) },
			func(reason object.Object) { done.Resolve(outcome(source, reason)) },
		)
		return done
	}
	return evaluator.NULL
}

// targascriptObject is the `targascript` value scripts see. Its run function
// runs another program in a fresh environment, as the playground does in a
// worker, and returns the error message or null. session() makes an
// environment that keeps its variables between runs, for a REPL: its run
// does the same, and its eval returns the value of the code, or fails with
// the error message.
func targascriptObject() *object.Hash {
	hash := object.NewHash()
	hash.Set(&object.String{Value: "run"}, runner("targascript.run", nil, false))
	hash.Set(&object.String{Value: "session"}, &object.Builtin{
		Name: "session",
		Fn: func(args ...object.Object) object.Object {
			s := newSession()
			session := object.NewHash()
			session.Set(&object.String{Value: "run"}, runner("session.run", s, false))
			session.Set(&object.String{Value: "eval"}, runner("session.eval", s, true))
			return session
		},
	})
	return hash
}

// runner is a builtin that runs code, given with a function for its output,
// in s, or in a fresh environment each time when s is nil. With value set
// it returns the value of the code, instead of null or the error message.
func runner(name string, s *session, value bool) *object.Builtin {
	return &object.Builtin{
		Name: name,
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Message: name + " takes the code and a function for its output"}
			}
			source, ok := args[0].(*object.String)
			if !ok || !evaluator.IsCallable(args[1]) {
				return &object.Error{Message: name + " takes the code, as a string, and a function for its output"}
			}
			out := writer(func(text string) {
				evaluator.CallFunction(args[1], []object.Object{&object.String{Value: text}})
			})
			target := s
			if target == nil {
				target = newSession()
			}
			if !value {
				return run(source.Value, target, out)
			}
			return evalValue(source.Value, target, out)
		},
	}
}

// evaluate runs the string with handle source, sending output to the
// JavaScript function onOutput. With fresh set it uses a new environment,
// otherwise the page's. dir is the directory the script came from, for load. It returns the handle of the error message, or 0
// (undefined).
//
//go:wasmexport evaluate
func evaluate(source, onOutput, fresh, dir uint32) uint32 {
	s := page
	if fresh != 0 {
		s = newSession()
	}
	// Load paths are relative to the script's directory on the server
	s.compiler.Dir = jsbridge.ReadString(dir)
	// Output can arrive after this returns, when the program awaits
	fn := jsbridge.Keep(onOutput)
	out := writer(func(text string) {
		fn.Call([]object.Object{&object.String{Value: text}})
	})
	return jsbridge.Return(run(jsbridge.ReadString(source), s, out))
}

// check finds the errors in the string with handle source without running
// it: syntax errors, or the problems the compiler knows about, like names
// that aren't defined. It returns the handle of a JSON array of objects with
// line, column and message.
//
//go:wasmexport check
func check(source uint32) uint32 {
	text := jsbridge.ReadString(source)
	problems := &object.Array{}
	add := func(line, column int, message string) {
		problem := object.NewHash()
		problem.Set(&object.String{Value: "line"}, &object.Integer{Value: int64(line)})
		problem.Set(&object.String{Value: "column"}, &object.Integer{Value: int64(column)})
		problem.Set(&object.String{Value: "message"}, &object.String{Value: message})
		problems.Elements = append(problems.Elements, problem)
	}

	p := parser.New(lexer.New(text))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		for _, e := range p.Diagnostics() {
			add(e.Line, e.Column, e.Message)
		}
	} else {
		s := newSession()
		if _, err := s.compiler.Compile(program); err != nil {
			add(0, 0, err.Error())
		}
		for _, problem := range s.compiler.Problems() {
			add(problem.Line, problem.Column, problem.Message)
		}
	}

	toJSON, _ := evaluator.Builtin("toJSON")
	return jsbridge.Return(toJSON.Fn(problems))
}

// formatSource returns the string with handle source laid out like
// targascript fmt does, as a handle.
//
//go:wasmexport format
func formatSource(source uint32) uint32 {
	return jsbridge.Return(&object.String{Value: format.Source(jsbridge.ReadString(source))})
}

// callback runs a TargaScript function passed to JavaScript.
//
//go:wasmexport callback
func callback(id, args uint32) uint32 {
	return jsbridge.Callback(id, args)
}

func init() {
	vm.MaxNestedRuns = maxNestedRuns
	compiler.ReadFile = readFile
	evaluator.SetHost(jsbridge.Host{})
	evaluator.SetRegexEngine(jsbridge.Regex{})
	evaluator.SetMathHost(jsbridge.Math{})
	evaluator.SetTimeHost(jsbridge.Time{})
	evaluator.SetOutput(writer(toConsole))
	page = newSession()
}

func main() {}

// readFile returns a file a program loads, which targascript.js fetched from
// the server before running the program.
func readFile(path string) ([]byte, error) {
	if content, ok := jsbridge.Global().CallMethod("targascriptReadFile", []object.Object{&object.String{Value: path}}).(*object.String); ok {
		return []byte(content.Value), nil
	}
	return nil, compiler.ErrNotExist
}
