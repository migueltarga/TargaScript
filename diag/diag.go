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

	// Trace is the calls that led to a runtime error, the innermost first
	Trace []Call
}

// Call is a function in a trace and the line it was at. The top level of a
// program has no name.
type Call struct {
	Function string
	Line     int
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
	writeTrace(&b, d.Trace)
	return b.String()
}

// writeTrace writes the calls that led to an error, when it happened inside
// a function:
//
//	in divide (line 2)
//	called from average (line 6)
//	called from line 10
func writeTrace(b *strings.Builder, trace []Call) {
	if len(trace) < 2 {
		return
	}

	// The same call over and over, like in runaway recursion, is one line
	type group struct {
		call  Call
		times int
	}
	groups := []group{}
	for _, call := range trace {
		if n := len(groups); n > 0 && groups[n-1].call == call {
			groups[n-1].times++
			continue
		}
		groups = append(groups, group{call, 1})
	}

	// A long trace keeps its innermost and outermost calls
	const shownFirst, shownLast = 8, 3
	skipped := 0
	if len(groups) > shownFirst+shownLast+1 {
		for _, g := range groups[shownFirst : len(groups)-shownLast] {
			skipped += g.times
		}
		groups = append(groups[:shownFirst:shownFirst], groups[len(groups)-shownLast:]...)
	}

	b.WriteString("\n")
	for i, g := range groups {
		if i == shownFirst && skipped > 0 {
			fmt.Fprintf(b, "\n  ... %d more calls", skipped)
		}
		b.WriteString("\n  ")
		if i == 0 {
			b.WriteString("in ")
		} else {
			b.WriteString("called from ")
		}
		if g.call.Function == "" {
			fmt.Fprintf(b, "line %d", g.call.Line)
		} else {
			fmt.Fprintf(b, "%s (line %d)", g.call.Function, g.call.Line)
		}
		if g.times > 1 {
			fmt.Fprintf(b, ", %d times", g.times)
		}
	}
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
