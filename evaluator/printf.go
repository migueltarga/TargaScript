package evaluator

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/migueltarga/TargaScript/object"
)

// formatPrintf formats args with Go's format verbs, checking that there is
// one argument of a suitable type for each verb. Go itself prints markers
// such as %!d(MISSING) into the output instead of failing.
func formatPrintf(format string, args []object.Object) (string, *object.Error) {
	var values []interface{}
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
			continue
		}
		i++
		if i < len(format) && format[i] == '%' {
			continue
		}

		for i < len(format) && strings.IndexByte("+-# 0", format[i]) >= 0 {
			i++
		}
		for _, part := range []string{"width", "precision"} {
			if part == "precision" {
				if i >= len(format) || format[i] != '.' {
					break
				}
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
						"printf %s given by * must be an INTEGER, got %s", part, arg.Type())}
				}
				values = append(values, int(n.Value))
				i++
				continue
			}
			for i < len(format) && format[i] >= '0' && format[i] <= '9' {
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
		values = append(values, value)
	}

	if next < len(args) {
		return "", &object.Error{Message: fmt.Sprintf(
			"too many arguments for printf: format uses %d of the %d given", next, len(args))}
	}

	return fmt.Sprintf(format, values...), nil
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
		"printf verb %%%c does not accept %s", verb, arg.Type())}
}
