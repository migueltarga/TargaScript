package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/parser"
	"github.com/migueltarga/TargaScript/repl"
)

func main() {
	noColor := flag.Bool("no-color", false, "Disable colored output")

	flag.Parse()

	fmt.Printf("Welcome to TargaScript v0.0.1\n")

	args := flag.Args()
	if len(args) > 0 {
		// Run file provided as argument
		filename := args[0]

		// Check file extension
		ext := filepath.Ext(filename)
		if ext != ".tg" {
			fmt.Printf("Warning: '%s' does not have the .tg extension. TargaScript files should use the .tg extension.\n", filename)
		}

		input, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("Error reading file: %s\n", err)
			os.Exit(1)
		}

		l := lexer.New(string(input))
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			fmt.Println("Parser errors:")
			for _, msg := range p.Errors() {
				fmt.Printf("\t%s\n", msg)
			}
			os.Exit(1)
		}

		fmt.Printf("File '%s' parsed successfully!\n", filename)
		fmt.Println("AST Structure:")

		if *noColor {
			fmt.Println(ast.NoColorPrettyPrint(program))
		} else {
			ast.PrettyPrintToTerminal(program)
		}
		return
	}
	fmt.Println("")
	fmt.Println("Type commands and press Enter to execute")
	fmt.Println("Press Ctrl+C to exit")

	repl.Start(os.Stdin, os.Stdout, *noColor)
}
