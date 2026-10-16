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

// mainDir is the directory load paths are relative to in the main program.
var mainDir string

// SetMainDir sets the directory used to resolve `load` paths in the main
// program, normally the directory of the script being run. Empty means the
// working directory.
func SetMainDir(dir string) {
	mainDir = dir
}

func EvalSource(source string, out io.Writer, noColor bool, debug bool, isRepl bool) error {
	return evalSource(source, newSession(), out, noColor, debug, isRepl)
}

// session keeps the variables of a program between REPL lines.
type session struct {
	compiler *compiler.Compiler
	machine  *vm.VM
}

func newSession() *session {
	c := compiler.New()
	c.Dir = mainDir
	s := &session{compiler: c, machine: vm.New(c.Globals())}
	s.machine.SetGlobal("NULL", evaluator.NULL)
	return s
}

// run compiles and runs a parsed program, returning its value or error.
func (s *session) run(program *ast.Program) object.Object {
	bytecode, err := s.compiler.Compile(program)
	if err != nil {
		return &object.Error{Message: err.Error()}
	}
	return s.machine.Run(bytecode)
}

func evalSource(source string, sess *session, out io.Writer, noColor bool, debug bool, isRepl bool) error {
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

	evaluator.SetOutput(out)

	var evaluated object.Object
	func() {
		defer func() {
			if r := recover(); r != nil {
				evaluated = &object.Error{Message: fmt.Sprintf("internal error: %v", r)}
			}
		}()

		evaluated = sess.run(program)
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

func Start(in io.Reader, out io.Writer, noColor bool, debug bool) {
	scanner := bufio.NewScanner(in)
	sess := newSession()

	for {
		fmt.Fprint(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		evalSource(line, sess, out, noColor, debug, true)
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
