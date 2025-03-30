package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/migueltarga/TargaScript/repl"
)

func main() {
	debug := flag.Bool("debug", false, "Enable debug mode with AST pretty printing (colored)")
	debugNoColor := flag.Bool("debug-no-color", false, "Enable debug mode with AST pretty printing (no colors)")

	flag.Parse()

	// Calculate if we should use colors based on which debug flag was used
	noColor := *debugNoColor
	debugEnabled := *debug || *debugNoColor

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

		fmt.Printf("Executing file '%s'...\n", filename)

		// Use the common evaluation function from repl package
		err = repl.EvalSource(string(input), os.Stdout, noColor, debugEnabled)
		if err != nil {
			os.Exit(1)
		}

		return
	}

	fmt.Println("")
	fmt.Println("Type commands and press Enter to execute")
	fmt.Println("Press Ctrl+C to exit")

	repl.Start(os.Stdin, os.Stdout, noColor, debugEnabled)
}
