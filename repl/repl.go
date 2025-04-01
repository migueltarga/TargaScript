package repl

import (
	"bufio"
	"fmt"
	"io"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
)

const PROMPT = ">> "

func EvalSource(source string, out io.Writer, noColor bool, debug bool, isRepl bool, trace bool) error {
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		printParserErrors(out, p.Errors())
		return fmt.Errorf("parser errors")
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

	env := object.NewEnvironment()
	env.Set("NULL", evaluator.NULL)

	var evaluated object.Object
	func() {
		defer func() {
			if r := recover(); r != nil {
				errMsg := fmt.Sprintf("Runtime panic: %v", r)
				if isRepl {
					fmt.Fprintf(out, "\033[31m%s\033[0m\n", errMsg)
				}
				evaluated = &object.Error{
					Message: errMsg,
					Line:    0,
					Column:  0,
				}
			}
		}()

		evaluated = evaluator.Eval(program, env)

		// Check if we got an error and print it only in REPL mode
		if isRepl && evaluated != nil && evaluated.Type() == object.ERROR_OBJ {
			errObj := evaluated.(*object.Error)
			fmt.Fprintf(out, "\033[31mRuntime Error: %s (line %d, column %d)\033[0m\n",
				errObj.Message, errObj.Line, errObj.Column)
		}
	}()

	if isRepl && evaluated != nil && evaluated.Type() != object.NULL_OBJ &&
		evaluated.Type() != object.ERROR_OBJ {
		io.WriteString(out, evaluated.Inspect())
		io.WriteString(out, "\n")
	}

	// If the evaluation resulted in an error, return it
	if evaluated != nil && evaluated.Type() == object.ERROR_OBJ {
		errObj := evaluated.(*object.Error)
		return fmt.Errorf("%s (line %d, column %d)",
			errObj.Message, errObj.Line, errObj.Column)
	}

	return nil
}

func Start(in io.Reader, out io.Writer, noColor bool, debug bool, trace bool) {
	scanner := bufio.NewScanner(in)

	for {
		fmt.Fprint(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		EvalSource(line, out, noColor, debug, true, trace)
	}
}

func printParserErrors(out io.Writer, errors []string) {
	io.WriteString(out, "Parser errors:\n")
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}
