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

	// Calculate color and debug settings
	noColor := *debugNoColor
	debugEnabled := *debug || *debugNoColor

	args := flag.Args()
	if len(args) > 0 {
		// Run file provided as argument
		filename := args[0]

		// Check file extension
		ext := filepath.Ext(filename)
		if ext != ".tg" && debugEnabled {
			fmt.Printf("Warning: '%s' does not have the .tg extension. TargaScript files should use the .tg extension.\n", filename)
		}

		if abs, err := filepath.Abs(filename); err == nil {
			repl.SetMainDir(filepath.Dir(abs))
		}

		input, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("Error reading file: %s\n", err)
			os.Exit(1)
		}

		if debugEnabled {
			fmt.Printf("Welcome to TargaScript v0.0.1\n")
			fmt.Printf("Executing file '%s'...\n", filename)
		}

		err = executeFile(string(input), debugEnabled, noColor)
		if err != nil {
			os.Exit(1)
		}

		return
	}

	fmt.Printf("Welcome to TargaScript v0.0.1\n")
	fmt.Println("")
	fmt.Println("Type commands and press Enter to execute")
	fmt.Println("Press Ctrl+C to exit")

	repl.Start(os.Stdin, os.Stdout, noColor, debugEnabled)
}

func executeFile(input string, debug bool, noColor bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Panic during execution: %v\n", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	err = repl.EvalSource(input, os.Stdout, noColor, debug, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m%v\033[0m\n", err)
		return err
	}
	return nil
}
