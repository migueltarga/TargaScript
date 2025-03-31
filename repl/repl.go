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

func EvalSource(source string, out io.Writer, noColor bool, debug bool, isRepl bool) error {
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

	evaluator.SetOutput(out)

	env := object.NewEnvironment()
	evaluated := evaluator.Eval(program, env)

	// Only print the final result in REPL mode
	if isRepl && evaluated != nil && evaluated.Type() != object.NULL_OBJ {
		io.WriteString(out, evaluated.Inspect())
		io.WriteString(out, "\n")
	}

	return nil
}

func Start(in io.Reader, out io.Writer, noColor bool, debug bool) {
	scanner := bufio.NewScanner(in)

	for {
		fmt.Fprint(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		EvalSource(line, out, noColor, debug, true)
	}
}

func printParserErrors(out io.Writer, errors []string) {
	io.WriteString(out, "Parser errors:\n")
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}
