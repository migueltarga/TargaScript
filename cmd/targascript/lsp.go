package main

import (
	"os"

	"github.com/migueltarga/TargaScript/lsp"
	"github.com/migueltarga/TargaScript/repl"
)

// runLSP serves the Language Server Protocol on standard input and output,
// for editors.
func runLSP() int {
	if err := lsp.NewServer(os.Stdout, repl.Check).Serve(os.Stdin); err != nil {
		return 1
	}
	return 0
}
