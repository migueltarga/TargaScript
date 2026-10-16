package repl

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/compiler"
	"github.com/migueltarga/TargaScript/diag"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
	"github.com/migueltarga/TargaScript/vm"
)

const PROMPT = ">> "

func EvalSource(source string, out io.Writer, noColor bool, debug bool, isRepl bool, trace bool) error {
	return evalSource(source, newSession(), out, noColor, debug, isRepl, trace)
}

// session keeps the variables of a program between REPL lines. Code runs on
// the bytecode VM, or on the evaluator in trace mode, which only it supports.
type session struct {
	compiler *compiler.Compiler
	machine  *vm.VM
	env      *object.Environment
}

func newSession() *session {
	c := compiler.New()
	c.Dir = evaluator.MainDir()
	s := &session{compiler: c, machine: vm.New(c.Globals()), env: object.NewEnvironment()}

	s.env.Set("NULL", evaluator.NULL)
	s.machine.SetGlobal("NULL", evaluator.NULL)
	return s
}

// run evaluates a parsed program and returns its value or error.
func (s *session) run(program *ast.Program, trace bool) object.Object {
	if trace {
		return evaluator.Eval(program, s.env)
	}
	bytecode, err := s.compiler.Compile(program)
	if err != nil {
		return &object.Error{Message: err.Error()}
	}
	return s.machine.Run(bytecode)
}

func evalSource(source string, sess *session, out io.Writer, noColor bool, debug bool, isRepl bool, trace bool) error {
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		diagnostics := []diag.Diagnostic{}
		for _, e := range p.Diagnostics() {
			diagnostics = append(diagnostics, diag.Diagnostic{Line: e.Line, Column: e.Column, Message: e.Message})
		}
		err := errors.New(diag.FormatAll(source, diagnostics))
		if isRepl {
			printError(out, err, noColor)
		}
		return err
	}

	if debug {
		if noColor {
			io.WriteString(out, ast.NoColorPrettyPrint(program))
		} else {
			io.WriteString(out, ast.PrettyPrint(program))
		}
		io.WriteString(out, "\n")
	}

	evaluator.SetTraceMode(trace)
	evaluator.SetOutput(out)

	var evaluated object.Object
	func() {
		defer func() {
			if r := recover(); r != nil {
				evaluated = &object.Error{Message: fmt.Sprintf("internal error: %v", r)}
			}
		}()

		evaluated = sess.run(program, trace)
	}()

	if isRepl && evaluated != nil && evaluated.Type() != object.NULL_OBJ &&
		evaluated.Type() != object.ERROR_OBJ {
		io.WriteString(out, evaluated.Inspect())
		io.WriteString(out, "\n")
	}

	// If the evaluation resulted in an error, return it
	if errObj, ok := evaluated.(*object.Error); ok {
		err := errors.New(diag.Format(source, diag.Diagnostic{
			Line: errObj.Line, Column: errObj.Column, Message: errObj.Message,
		}))
		if isRepl {
			printError(out, err, noColor)
		}
		return err
	}

	return nil
}

func Start(in io.Reader, out io.Writer, noColor bool, debug bool, trace bool) {
	scanner := bufio.NewScanner(in)
	sess := newSession()

	for {
		fmt.Fprint(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		evalSource(line, sess, out, noColor, debug, true, trace)
	}
}

// printError writes an error, in red unless noColor is set.
func printError(out io.Writer, err error, noColor bool) {
	if noColor {
		fmt.Fprintln(out, err)
		return
	}
	fmt.Fprintf(out, "\033[31m%s\033[0m\n", err)
}
