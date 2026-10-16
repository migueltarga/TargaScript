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
	lines := strings.Split(strings.ReplaceAll(code, "\r\n", "\n"), "\n")
	infos := scan(lines)

	var out []string
	blank := 0
	for i, line := range lines {
		info := infos[i]
		if info.protected {
			out = append(out, line)
			blank = 0
			continue
		}
		text := strings.TrimLeft(line, " \t")
		if !info.open {
			text = strings.TrimRight(text, " \t")
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
)

// scan goes through the lines once, tracking strings with the code in
// their ${...}, comments and brackets. A line is indented one level more
// than the line that opened the innermost bracket still open, however many
// it opened: test("x", fn() { gives its body one level. A line that starts
// by closing a bracket gets the level of the line that opened it.
func scan(lines []string) []lineInfo {
	infos := make([]lineInfo, len(lines))
	// The levels of the lines that opened the brackets still open
	opened := []int{}
	// The modes entered, innermost last; braces inside ${...} are counted
	// per interpolation so they don't end it early
	stack := []mode{code}
	braces := []int{0}

	for i, line := range lines {
		current := stack[len(stack)-1]
		info := lineInfo{protected: current != code}
		text := strings.TrimLeft(line, " \t")

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
			switch stack[len(stack)-1] {
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
					if len(stack) > 1 {
						braces[len(braces)-1]++
					} else {
						opened = append(opened, indent)
					}
				case c == '}' || c == ')' || c == ']':
					if len(stack) > 1 {
						// The } that ends a ${...}
						if c == '}' && braces[len(braces)-1] == 0 {
							stack = stack[:len(stack)-1]
							braces = braces[:len(braces)-1]
							continue
						}
						braces[len(braces)-1]--
					} else if len(opened) > 0 {
						opened = opened[:len(opened)-1]
					}
				}
			case doubleQuoted, backticked:
				quote := byte('"')
				if stack[len(stack)-1] == backticked {
					quote = '`'
				}
				switch {
				case c == '\\':
					j++
				case c == quote:
					stack = stack[:len(stack)-1]
				case c == '$' && next == '{':
					stack = append(stack, code)
					braces = append(braces, 0)
					j++
				}
			case blockComment:
				if c == '*' && next == '/' {
					stack = stack[:len(stack)-1]
					j++
				}
			}
		}
		top := stack[len(stack)-1]
		info.open = top == doubleQuoted || top == backticked
		infos[i] = info
	}
	return infos
}
