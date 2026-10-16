package repl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/compiler"
	"github.com/migueltarga/TargaScript/diag"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/eventloop"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
	"github.com/migueltarga/TargaScript/vm"
)

const PROMPT = ">> "

// CONTINUE is the prompt for the next line of unfinished code, like a
// function whose body isn't closed yet.
const CONTINUE = ".. "

// mainDir is the directory load paths are relative to in the main program.
var mainDir string

// args are the command-line arguments after the script, which programs see
// as the global args.
var args = []string{}

// SetArgs sets the arguments programs see in args.
func SetArgs(values []string) {
	args = values
}

// loop runs the asynchronous work of builtins like sleep.
var loop = eventloop.New()

func init() {
	evaluator.SetHost(loop)
}

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
	list := &object.Array{Elements: []object.Object{}}
	for _, arg := range args {
		list.Elements = append(list.Elements, &object.String{Value: arg})
	}
	s.machine.SetGlobal("args", list)
	return s
}

// run compiles and runs a parsed program, returning its value or error.
func (s *session) run(program *ast.Program) object.Object {
	bytecode, err := s.compiler.Compile(program)
	if err != nil {
		return &object.Error{Message: err.Error()}
	}
	return wait(s.machine.Run(bytecode))
}

// wait runs the event loop until a program that paused at an await
// finishes, and returns its result. It then runs work still pending, like
// timers nothing waits for, as a script ends only when nothing is left to do.
func wait(result object.Object) object.Object {
	promise, ok := result.(*object.Promise)
	if !ok {
		loop.Run(func() bool { return false })
		return result
	}

	if !loop.Run(promise.Settled) {
		return &object.Error{Message: "the program is waiting for something that will never happen"}
	}
	loop.Run(func() bool { return false })

	value, rejected := promise.Result()
	if rejected {
		if err, ok := value.(*object.Error); ok {
			return err
		}
		return &object.Error{Message: evaluator.ToDisplayString(value)}
	}
	return value
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
	var code strings.Builder

	for {
		if code.Len() == 0 {
			fmt.Fprint(out, PROMPT)
		} else {
			fmt.Fprint(out, CONTINUE)
		}
		if !scanner.Scan() {
			if code.Len() > 0 {
				evalSource(code.String(), sess, out, noColor, debug, true)
			}
			return
		}

		line := scanner.Text()
		code.WriteString(line + "\n")
		if strings.TrimSpace(code.String()) == "" {
			code.Reset()
			continue
		}
		// Keep reading unfinished code; an empty line runs it anyway, to
		// show what is wrong
		if line != "" && unfinished(code.String()) {
			continue
		}
		evalSource(code.String(), sess, out, noColor, debug, true)
		code.Reset()
	}
}

// unfinished reports whether code stops in the middle of something, like a
// block or a string that isn't closed, or an operator with nothing after it.
func unfinished(code string) bool {
	p := parser.New(lexer.New(code))
	p.ParseProgram()
	for _, err := range p.Diagnostics() {
		if strings.Contains(err.Message, "never closed") || strings.Contains(err.Message, "the file ended") {
			return true
		}
		// An error right after the last character, like `let x = 1 +`
		lines := strings.Split(strings.TrimRight(code, "\n"), "\n")
		last := lines[len(lines)-1]
		if err.Line == len(lines) && err.Column > len([]rune(last)) {
			return true
		}
	}
	return false
}

// printError writes an error, in red unless noColor is set.
func printError(out io.Writer, err error, noColor bool) {
	if noColor {
		fmt.Fprintln(out, err)
		return
	}
	fmt.Fprintf(out, "\033[31m%s\033[0m\n", err)
}
