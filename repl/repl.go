package repl

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/diag"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
)

const PROMPT = ">> "

func EvalSource(source string, out io.Writer, noColor bool, debug bool, isRepl bool, trace bool) error {
	return evalSource(source, newEnvironment(), out, noColor, debug, isRepl, trace)
}

func newEnvironment() *object.Environment {
	env := object.NewEnvironment()
	env.Set("NULL", evaluator.NULL)
	return env
}

func evalSource(source string, env *object.Environment, out io.Writer, noColor bool, debug bool, isRepl bool, trace bool) error {
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

		evaluated = evaluator.Eval(program, env)
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
	env := newEnvironment()

	for {
		fmt.Fprint(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		evalSource(line, env, out, noColor, debug, true, trace)
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
