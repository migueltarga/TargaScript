package evaluator

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/migueltarga/TargaScript/object"
)

// formatPrintf formats args with Go's format verbs, checking that there is
// one argument of a suitable type for each verb. Go itself prints markers
// such as %!d(MISSING) into the output instead of failing. A , flag, which
// Go doesn't have, groups the thousands of a number: %,d and %,.2f.
func formatPrintf(format string, args []object.Object) (string, *object.Error) {
	var out strings.Builder
	next := 0

	take := func() (object.Object, *object.Error) {
		if next >= len(args) {
			return nil, &object.Error{Message: fmt.Sprintf(
				"not enough arguments for printf: format has more verbs than the %d given", len(args))}
		}
		next++
		return args[next-1], nil
	}

	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			out.WriteByte(format[i])
			continue
		}
		i++
		if i < len(format) && format[i] == '%' {
			out.WriteByte('%')
			continue
		}

		// The verb, written again without the , flag, and the numbers its
		// * take
		spec := "%"
		grouped, leftAligned := false, false
		var verbArgs []interface{}
		width := -1
		for i < len(format) && strings.IndexByte("+-# 0,", format[i]) >= 0 {
			switch format[i] {
			case ',':
				grouped = true
			case '-':
				leftAligned = true
				spec += "-"
			default:
				spec += string(format[i])
			}
			i++
		}
		for _, part := range []string{"width", "precision"} {
			if part == "precision" {
				if i >= len(format) || format[i] != '.' {
					break
				}
				spec += "."
				i++
			}
			if i < len(format) && format[i] == '*' {
				arg, err := take()
				if err != nil {
					return "", err
				}
				n, ok := arg.(*object.Integer)
				if !ok {
					return "", &object.Error{Message: fmt.Sprintf(
						"printf %s given by * must be an integer, got %s", part, typeName(arg))}
				}
				if part == "width" {
					width = int(n.Value)
				} else {
					spec += "*"
					verbArgs = append(verbArgs, int(n.Value))
				}
				i++
				continue
			}
			start := i
			for i < len(format) && format[i] >= '0' && format[i] <= '9' {
				i++
			}
			if part == "width" && i > start {
				width, _ = strconv.Atoi(format[start:i])
			} else {
				spec += format[start:i]
			}
			// The , can also follow the width, as in Python: %12,d
			if part == "width" && i < len(format) && format[i] == ',' {
				grouped = true
				i++
			}
		}

		if i >= len(format) {
			return "", &object.Error{Message: "printf format ends with an incomplete verb"}
		}
		verb, size := utf8.DecodeRuneInString(format[i:])
		i += size - 1

		arg, err := take()
		if err != nil {
			return "", err
		}
		value, err := printfValue(verb, arg)
		if err != nil {
			return "", err
		}
		text := fmt.Sprintf(spec+string(verb), append(verbArgs, value)...)
		if grouped {
			text = groupThousands(text)
		}
		// The width goes last, so it counts the commas
		for pad := width - utf8.RuneCountInString(text); pad > 0; pad-- {
			if leftAligned {
				text += " "
			} else if strings.Contains(spec, "0") && !grouped && isNumberText(text) {
				text = zeroPad(text)
			} else {
				text = " " + text
			}
		}
		out.WriteString(text)
	}

	if next < len(args) {
		return "", &object.Error{Message: fmt.Sprintf(
			"too many arguments for printf: format uses %d of the %d given", next, len(args))}
	}
	return out.String(), nil
}

// groupThousands puts commas between the thousands of the integer part of
// a formatted number: 1234567.5 becomes 1,234,567.5.
func groupThousands(text string) string {
	start := 0
	if start < len(text) && strings.IndexByte("+- ", text[start]) >= 0 {
		start++
	}
	end := start
	for end < len(text) && text[end] >= '0' && text[end] <= '9' {
		end++
	}
	digits := text[start:end]
	if len(digits) <= 3 {
		return text
	}
	var grouped strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			grouped.WriteByte(',')
		}
		grouped.WriteRune(d)
	}
	return text[:start] + grouped.String() + text[end:]
}

func isNumberText(text string) bool {
	return text != "" && strings.IndexByte("+-0123456789", text[0]) >= 0
}

// zeroPad adds one 0 after the sign of a number, as %05d does.
func zeroPad(text string) string {
	if text[0] == '+' || text[0] == '-' {
		return text[:1] + "0" + text[1:]
	}
	return "0" + text
}

// printfValue converts arg to the Go value the verb expects.
func printfValue(verb rune, arg object.Object) (interface{}, *object.Error) {
	switch verb {
	case 'v':
		switch arg := arg.(type) {
		case *object.Integer:
			return arg.Value, nil
		case *object.Float:
			return arg.Value, nil
		case *object.Boolean:
			return arg.Value, nil
		case *object.String:
			return arg.Value, nil
		}
		return arg.Inspect(), nil
	case 's', 'q':
		if s, ok := arg.(*object.String); ok {
			return s.Value, nil
		}
		return arg.Inspect(), nil
	case 'd', 'b', 'o', 'c':
		if n, ok := arg.(*object.Integer); ok {
			return n.Value, nil
		}
	case 'x', 'X':
		switch arg := arg.(type) {
		case *object.Integer:
			return arg.Value, nil
		case *object.String:
			return arg.Value, nil
		}
	case 'f', 'F', 'e', 'E', 'g', 'G':
		switch arg := arg.(type) {
		case *object.Integer:
			return float64(arg.Value), nil
		case *object.Float:
			return arg.Value, nil
		}
	case 't':
		if b, ok := arg.(*object.Boolean); ok {
			return b.Value, nil
		}
	default:
		return nil, &object.Error{Message: fmt.Sprintf("unknown printf verb %%%c", verb)}
	}

	return nil, &object.Error{Message: fmt.Sprintf(
		"printf verb %%%c does not accept %s", verb, typeName(arg))}
}
