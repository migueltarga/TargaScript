package repl

import (
	"bufio"
	"fmt"
	"io"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/parser"
)

const PROMPT = ">> "

// EvalSource evaluates a source string and returns any errors
// It can be used for both REPL and file execution
func EvalSource(source string, out io.Writer, noColor bool, debug bool) error {
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		printParserErrors(out, p.Errors())
		return fmt.Errorf("parser errors")
	}

	// Pretty print AST in debug mode
	if debug {
		if noColor {
			io.WriteString(out, ast.NoColorPrettyPrint(program))
		} else {
			io.WriteString(out, ast.PrettyPrint(program))
		}
		io.WriteString(out, "\n")
	}

	evaluated := evaluator.Eval(program)
	if evaluated != nil {
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
		// Use the common evaluation function
		EvalSource(line, out, noColor, debug)
	}
}

func printParserErrors(out io.Writer, errors []string) {
	io.WriteString(out, "Parser errors:\n")
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}
