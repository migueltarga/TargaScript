// Package diag formats errors with the line of code they point at.
package diag

import (
	"fmt"
	"strconv"
	"strings"
)

// Diagnostic is an error at a position in the source. A Line of 0 means
// the position is unknown.
type Diagnostic struct {
	Line    int
	Column  int
	Message string
}

// MaxShown is how many errors FormatAll shows before summarizing the rest.
const MaxShown = 3

// Format writes the error followed by its line of code, with a caret under
// the column:
//
//	Error: expected a value after "+"
//	  --> line 1, column 12
//
//	 1 | let x = 1 +
//	   |            ^
func Format(source string, d Diagnostic) string {
	var b strings.Builder
	b.WriteString("Error: " + d.Message)

	lines := strings.Split(source, "\n")
	if d.Line <= 0 || d.Line > len(lines) {
		if d.Line > 0 {
			fmt.Fprintf(&b, " (line %d, column %d)", d.Line, d.Column)
		}
		return b.String()
	}

	fmt.Fprintf(&b, "\n  --> line %d, column %d\n\n", d.Line, d.Column)

	text := strings.TrimRight(lines[d.Line-1], "\r")
	number := strconv.Itoa(d.Line)
	gutter := strings.Repeat(" ", len(number))
	fmt.Fprintf(&b, " %s | %s\n", number, text)
	fmt.Fprintf(&b, " %s | %s^", gutter, caretIndent(text, d.Column))
	return b.String()
}

// FormatAll formats the first MaxShown errors, separated by blank lines.
func FormatAll(source string, diagnostics []Diagnostic) string {
	parts := []string{}
	for i, d := range diagnostics {
		if i == MaxShown {
			rest := len(diagnostics) - MaxShown
			if rest == 1 {
				parts = append(parts, "...and 1 more error")
			} else {
				parts = append(parts, fmt.Sprintf("...and %d more errors", rest))
			}
			break
		}
		parts = append(parts, Format(source, d))
	}
	return strings.Join(parts, "\n\n")
}

// caretIndent returns the spacing that puts a caret under column, keeping
// the tabs of the line so the caret lines up however tabs are shown.
func caretIndent(text string, column int) string {
	var indent strings.Builder
	i := 1
	for _, r := range text {
		if i >= column {
			break
		}
		if r == '\t' {
			indent.WriteRune('\t')
		} else {
			indent.WriteRune(' ')
		}
		i++
	}
	for ; i < column; i++ {
		indent.WriteRune(' ')
	}
	return indent.String()
}
