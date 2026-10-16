package repl

import (
	"io"
	"os"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/term"
)

// terminalLines reads lines from a terminal with line editing: the arrow
// keys move through the line and through earlier lines.
type terminalLines struct {
	fd       int
	terminal *term.Terminal
}

// setNames makes Tab complete the name before the cursor with one of
// names: all of it when only one fits, or what the ones that fit share.
func (t terminalLines) setNames(names func() []string) {
	t.terminal.AutoCompleteCallback = func(line string, pos int, key rune) (string, int, bool) {
		if key != '\t' {
			return "", 0, false
		}
		start := pos
		for start > 0 && isNameRune(rune(line[start-1])) {
			start--
		}
		// A Tab at the start of a word is indentation
		if start == pos {
			return "", 0, false
		}
		completion := complete(line[start:pos], names())
		return line[:pos] + completion + line[pos:], pos + len(completion), true
	}
}

// complete returns what to add to prefix: the rest of the only name that
// starts with it, or the longest start the names that do have in common.
func complete(prefix string, names []string) string {
	matches := []string{}
	for _, name := range names {
		if strings.HasPrefix(name, prefix) && name != prefix {
			matches = append(matches, name)
		}
	}
	if len(matches) == 0 {
		return ""
	}
	sort.Strings(matches)
	shared := matches[0]
	for _, m := range matches[1:] {
		for !strings.HasPrefix(m, shared) {
			shared = shared[:len(shared)-1]
		}
	}
	return shared[len(prefix):]
}

func isNameRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
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
