package repl

import (
	"io"
	"os"

	"golang.org/x/term"
)

// terminalLines reads lines from a terminal with line editing: the arrow
// keys move through the line and through earlier lines.
type terminalLines struct {
	fd       int
	terminal *term.Terminal
}

func (t terminalLines) readLine(prompt string) (string, bool) {
	// The terminal is raw only while a line is typed, so programs that
	// print or call input see it as usual
	state, err := term.MakeRaw(t.fd)
	if err != nil {
		return "", false
	}
	defer term.Restore(t.fd, state)

	t.terminal.SetPrompt(prompt)
	line, err := t.terminal.ReadLine()
	if err == term.ErrPasteIndicator {
		err = nil
	}
	return line, err == nil
}

// StartTerminal runs the REPL on standard input, with line editing and
// history when it's a terminal.
func StartTerminal(noColor bool, debug bool) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		Start(os.Stdin, os.Stdout, noColor, debug)
		return
	}
	terminal := term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}, PROMPT)
	run(terminalLines{fd, terminal}, os.Stdout, noColor, debug)
}
