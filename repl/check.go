package repl

import (
	"github.com/migueltarga/TargaScript/diag"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/parser"
)

// Check finds the problems in a program without running it: syntax errors,
// or the ones the compiler knows about, like names that aren't defined.
// dir is the directory its loads are relative to.
func Check(source, dir string) []diag.Diagnostic {
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	problems := []diag.Diagnostic{}
	if len(p.Errors()) > 0 {
		for _, e := range p.Diagnostics() {
			problems = append(problems, diag.Diagnostic{Line: e.Line, Column: e.Column, Message: e.Message})
		}
		return problems
	}

	s := newSession()
	s.compiler.Dir = dir
	// The globals of the browser, so scripts for web pages check too
	s.machine.SetGlobal("js", evaluator.NULL)
	s.machine.SetGlobal("targascript", evaluator.NULL)
	if _, err := s.compiler.Compile(program); err != nil {
		problems = append(problems, diag.Diagnostic{Message: err.Error()})
	}
	for _, problem := range s.compiler.Problems() {
		problems = append(problems, diag.Diagnostic{Line: problem.Line, Column: problem.Column, Message: problem.Message})
	}
	return problems
}
