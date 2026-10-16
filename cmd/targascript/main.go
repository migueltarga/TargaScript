package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/pprof"

	"github.com/migueltarga/TargaScript/compiler"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/repl"
)

// version is set by the release build.
var version = "dev"

func main() {
	// An executable made by targascript build runs its program, and every
	// argument is the program's
	if b, ok := readBundle(); ok {
		os.Exit(runBundle(b))
	}

	// TARGASCRIPT_CPUPROFILE=file writes a CPU profile of the run, for
	// bench/pgo.sh and for finding what makes a program slow
	if path := os.Getenv("TARGASCRIPT_CPUPROFILE"); path != "" {
		if file, err := os.Create(path); err == nil && pprof.StartCPUProfile(file) == nil {
			defer pprof.StopCPUProfile()
		}
	}

	showVersion := flag.Bool("version", false, "Print the version and exit")
	code := flag.String("e", "", "Run the given code instead of a file")
	debug := flag.Bool("debug", false, "Enable debug mode with AST pretty printing (colored)")
	debugNoColor := flag.Bool("debug-no-color", false, "Enable debug mode with AST pretty printing (no colors)")
	watch := flag.Bool("watch", false, "Run the file again whenever a .tg file next to it changes")

	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Println("targascript", version)
		return
	}

	// Calculate color and debug settings
	noColor := *debugNoColor
	debugEnabled := *debug || *debugNoColor

	args := flag.Args()

	// test --watch, as well as --watch test
	if len(args) > 0 && args[0] == "test" && (contains(args, "--watch") || contains(args, "-watch")) {
		*watch = true
	}
	if *watch {
		if len(args) == 0 || *code != "" {
			fmt.Fprintln(os.Stderr, "--watch needs a file to run, or test")
			os.Exit(1)
		}
		dir := filepath.Dir(args[0])
		if args[0] == "test" {
			dir = "."
		}
		os.Exit(runWatch(withoutWatch(os.Args[1:]), dir))
	}

	if *code != "" {
		repl.SetArgs(args)
		if executeFile(*code, debugEnabled, noColor) != nil {
			os.Exit(1)
		}
		return
	}

	if len(args) > 0 && args[0] == "help" {
		if _, err := os.Stat("help"); err != nil {
			usage()
			return
		}
	}

	// test, check and build are commands, unless there's a file with
	// their name
	if len(args) > 0 && (args[0] == "test" || args[0] == "check" || args[0] == "build" || args[0] == "lsp" || args[0] == "fmt") {
		if _, err := os.Stat(args[0]); err != nil {
			switch args[0] {
			case "test":
				os.Exit(runTests(args[1:], noColor))
			case "check":
				os.Exit(runCheck(args[1:]))
			case "lsp":
				os.Exit(runLSP())
			case "fmt":
				os.Exit(runFmt(args[1:]))
			default:
				os.Exit(runBuild(args[1:]))
			}
		}
	}

	// - reads the program from standard input
	if len(args) > 0 && args[0] == "-" {
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Printf("Error reading standard input: %s\n", err)
			os.Exit(1)
		}
		repl.SetArgs(args[1:])
		if executeFile(string(input), debugEnabled, noColor) != nil {
			os.Exit(1)
		}
		return
	}

	if len(args) > 0 {
		// Run file provided as argument, passing it the arguments after it
		filename := args[0]
		repl.SetArgs(args[1:])

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
			fmt.Printf("Welcome to TargaScript %s\n", version)
			fmt.Printf("Executing file '%s'...\n", filename)
		}

		err = executeFile(string(input), debugEnabled, noColor)
		if err != nil {
			os.Exit(1)
		}

		return
	}

	fmt.Printf("Welcome to TargaScript %s\n", version)
	fmt.Println("")
	fmt.Println("Type commands and press Enter to execute")
	fmt.Println("Type help for help, exit or Ctrl+D to leave")

	repl.StartTerminal(noColor, debugEnabled)
}

func executeFile(input string, debug bool, noColor bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Panic during execution: %v\n", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	out := scriptOutput()
	defer evaluator.FlushOutput()
	err = repl.EvalSource(input, out, noColor, debug, false)
	evaluator.FlushOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m%v\033[0m\n", err)
		return err
	}
	return nil
}

// runBundle runs the program of a built executable, reading the files it
// loads from the bundle, and returns the exit code.
func runBundle(b *bundle) int {
	fallback := compiler.ReadFile
	compiler.ReadFile = func(path string) ([]byte, error) {
		if source, ok := b.Files[path]; ok {
			return []byte(source), nil
		}
		return fallback(path)
	}
	repl.SetArgs(os.Args[1:])
	repl.SetMainDir(filepath.Dir(b.Main))
	if executeFile(b.Files[b.Main], false, false) != nil {
		return 1
	}
	return 0
}

// usage prints how to use the command, for -h, --help and help.
func usage() {
	fmt.Fprint(os.Stderr, `TargaScript, a small scripting language.

Usage:
  targascript                      start the REPL
  targascript file.tg [args...]    run a program; it sees the arguments in args
  targascript --watch file.tg      run it again whenever a .tg file next to it changes
  targascript -e 'code'            run code given on the command line
  targascript -                    run a program read from standard input

Commands:
  targascript test [-run text] [--watch] [paths]
                                   run the tests in *_test.tg files; -run only those named like text,
                                   --watch again whenever a .tg file changes
  targascript check [paths]        find problems in .tg files without running them
  targascript fmt [--check] [paths]
                                   lay out .tg files; --check only lists the ones that need it
  targascript build file.tg [-o name]
                                   make an executable with the program and what it loads
  targascript lsp                  a language server, for editors

Flags:
`)
	flag.PrintDefaults()
	fmt.Fprint(os.Stderr, `
Reference: https://github.com/migueltarga/TargaScript/blob/develop/docs/REFERENCE.md
Playground: https://script.targa.me/
`)
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

// withoutWatch returns the arguments without --watch, for the runs of watch.
func withoutWatch(args []string) []string {
	var out []string
	for _, arg := range args {
		if arg != "--watch" && arg != "-watch" {
			out = append(out, arg)
		}
	}
	return out
}
