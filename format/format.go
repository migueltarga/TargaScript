// Package format lays out TargaScript code: it indents each line by the
// blocks around it, removes spaces at the ends of lines and extra blank
// lines. It doesn't change what's inside a line, nor strings and comments
// that span lines, so formatting never changes what a program does.
package format

import "strings"

// Indent is one level of indentation.
const Indent = "    "

// lineInfo is what Source needs to know about a line before laying it out.
type lineInfo struct {
	indent    int  // levels of indentation
	protected bool // it starts inside a string or a block comment
	open      bool // it ends inside a string, so its end is text
}

// Source returns code laid out.
func Source(code string) string {
	// The lexer stops at a NUL byte, which text files don't have
	if strings.IndexByte(code, 0) >= 0 {
		return code
	}
	// Line breaks become \n, except inside strings, where \r\n is text:
	// the \r of other lines goes with the spaces at their end
	lines := strings.Split(code, "\n")
	infos, unclosed := scan(lines)
	// A string or a comment that never ends would take in what formatting
	// adds, so code that broken is left as it is
	if unclosed {
		return code
	}

	var out []string
	blank := 0
	for i, line := range lines {
		info := infos[i]
		if info.protected {
			out = append(out, line)
			blank = 0
			continue
		}
		text := strings.TrimLeft(line, " \t\r")
		if !info.open {
			text = strings.TrimRight(text, " \t\r")
		}
		if text == "" {
			blank++
			// One blank line at most, and none at the start
			if blank > 1 || len(out) == 0 {
				continue
			}
			out = append(out, "")
			continue
		}
		blank = 0
		out = append(out, strings.Repeat(Indent, info.indent)+text)
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "\n") + "\n"
}

// continues reports whether a line goes on with the expression of the line
// before it, like `|> next` or `.map(...)`, which is indented one level more.
func continues(text string) bool {
	for _, start := range []string{"|>", "?.", "&&", "||", "??"} {
		if strings.HasPrefix(text, start) {
			return true
		}
	}
	return strings.HasPrefix(text, ".") && !strings.HasPrefix(text, "...") && !strings.HasPrefix(text, "..")
}

// mode is where the scanner is: code, inside a string, or a comment.
type mode int

const (
	code mode = iota
	doubleQuoted
	backticked
	blockComment
	// The code of a ${...}, read the way the lexer skips it: only braces
	// and double-quoted strings count there
	interpolation
	quotedInInterpolation
)

// scan goes through the lines once, tracking strings with the code in
// their ${...}, comments and brackets, the way the lexer reads them. A line
// is indented one level more than the line that opened the innermost
// bracket still open, however many it opened: test("x", fn() { gives its
// body one level. A line that starts by closing a bracket gets the level of
// the line that opened it. unclosed reports a string or a comment that
// never ends.
func scan(lines []string) (infos []lineInfo, unclosed bool) {
	infos = make([]lineInfo, len(lines))
	// The levels of the lines that opened the brackets still open
	opened := []int{}
	// The modes entered, innermost last, and the braces open in each
	// interpolation
	stack := []mode{code}
	braces := []int{}

	for i, line := range lines {
		// Inside a string, even in its ${...}, a line is part of the string
		info := lineInfo{protected: len(stack) > 1}
		text := strings.TrimLeft(line, " \t\r")

		indent := 0
		if n := len(opened); n > 0 {
			indent = opened[n-1] + 1
		}
		if len(stack) == 1 && text != "" && strings.ContainsRune(")]}", rune(text[0])) && len(opened) > 0 {
			indent = opened[len(opened)-1]
		} else if len(stack) == 1 && continues(text) {
			indent++
		}
		info.indent = indent

		for j := 0; j < len(line); j++ {
			c := line[j]
			next := byte(0)
			if j+1 < len(line) {
				next = line[j+1]
			}
			top := stack[len(stack)-1]
			switch top {
			case code:
				switch {
				case c == '/' && next == '/':
					j = len(line)
				case c == '/' && next == '*':
					stack = append(stack, blockComment)
					j++
				case c == '"':
					stack = append(stack, doubleQuoted)
				case c == '`':
					stack = append(stack, backticked)
				case c == '{' || c == '(' || c == '[':
					opened = append(opened, indent)
				case c == '}' || c == ')' || c == ']':
					if len(opened) > 0 {
						opened = opened[:len(opened)-1]
					}
				}
			case doubleQuoted, backticked:
				quote := byte('"')
				if top == backticked {
					quote = '`'
				}
				switch {
				// In backticks only \` and \$ are escapes, as in the lexer
				case c == '\\' && (quote == '"' || next == '`' || next == '$'):
					j++
				case c == quote:
					stack = stack[:len(stack)-1]
				case c == '$' && next == '{':
					stack = append(stack, interpolation)
					braces = append(braces, 0)
					j++
				}
			case interpolation:
				switch c {
				case '{':
					braces[len(braces)-1]++
				case '}':
					if braces[len(braces)-1] == 0 {
						stack = stack[:len(stack)-1]
						braces = braces[:len(braces)-1]
					} else {
						braces[len(braces)-1]--
					}
				case '"':
					stack = append(stack, quotedInInterpolation)
				}
			case quotedInInterpolation:
				switch c {
				case '\\':
					j++
				case '"':
					stack = stack[:len(stack)-1]
				}
			case blockComment:
				if c == '*' && next == '/' {
					stack = stack[:len(stack)-1]
					j++
				}
			}
		}
		top := stack[len(stack)-1]
		info.open = top != code && top != blockComment
		infos[i] = info
	}
	return infos, len(stack) > 1
}
