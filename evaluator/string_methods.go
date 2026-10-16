package evaluator

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/strfmt"
)

// maxStringLength limits the strings repeat and the pad methods build, so a
// typo like "x".repeat(1e12) fails instead of exhausting memory.
const maxStringLength = 100_000_000

// moreStringMethods are string methods beyond the ones in
// evalMethodCallExpression. Positions are in characters, like indexing.
var moreStringMethods = map[string]func(s string, args []object.Object) object.Object{
	"repeat": func(s string, args []object.Object) object.Object {
		n, err := intArgs("repeat", args, 1, 1)
		if err != nil {
			return err
		}
		if n[0] < 0 {
			return &object.Error{Message: "string.repeat() needs a count of 0 or more"}
		}
		if int64(len(s))*n[0] > maxStringLength {
			return &object.Error{Message: "string.repeat() would make a string too long"}
		}
		return &object.String{Value: strings.Repeat(s, int(n[0]))}
	},
	"padStart": func(s string, args []object.Object) object.Object {
		return pad("padStart", s, args, true)
	},
	"padEnd": func(s string, args []object.Object) object.Object {
		return pad("padEnd", s, args, false)
	},
	"reverse": func(s string, args []object.Object) object.Object {
		if err := noArgs("reverse", args); err != nil {
			return err
		}
		runes := []rune(s)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return &object.String{Value: string(runes)}
	},
	"slice": func(s string, args []object.Object) object.Object {
		runes := []rune(s)
		bounds, err := intArgs("slice", args, 1, 2)
		if err != nil {
			return err
		}
		length := int64(len(runes))
		start := clampSliceIndex(bounds[0], length)
		end := length
		if len(bounds) == 2 {
			end = clampSliceIndex(bounds[1], length)
		}
		if start > end {
			start = end
		}
		return &object.String{Value: string(runes[start:end])}
	},
	"chars": func(s string, args []object.Object) object.Object {
		if err := noArgs("chars", args); err != nil {
			return err
		}
		chars := []object.Object{}
		for _, r := range s {
			chars = append(chars, &object.String{Value: string(r)})
		}
		return &object.Array{Elements: chars}
	},
	"lines": func(s string, args []object.Object) object.Object {
		if err := noArgs("lines", args); err != nil {
			return err
		}
		lines := []object.Object{}
		if s == "" {
			return &object.Array{Elements: lines}
		}
		for _, line := range strings.Split(strings.TrimSuffix(s, "\n"), "\n") {
			lines = append(lines, &object.String{Value: strings.TrimSuffix(line, "\r")})
		}
		return &object.Array{Elements: lines}
	},
	"trimStart": func(s string, args []object.Object) object.Object {
		if err := noArgs("trimStart", args); err != nil {
			return err
		}
		return &object.String{Value: strings.TrimLeftFunc(s, unicode.IsSpace)}
	},
	"trimEnd": func(s string, args []object.Object) object.Object {
		if err := noArgs("trimEnd", args); err != nil {
			return err
		}
		return &object.String{Value: strings.TrimRightFunc(s, unicode.IsSpace)}
	},
	"count": func(s string, args []object.Object) object.Object {
		sub, err := stringArg("count", args)
		if err != nil {
			return err
		}
		if sub == "" {
			return &object.Error{Message: "string.count() needs a non-empty string to count"}
		}
		return &object.Integer{Value: int64(strings.Count(s, sub))}
	},
	"capitalize": func(s string, args []object.Object) object.Object {
		if err := noArgs("capitalize", args); err != nil {
			return err
		}
		for i, r := range s {
			return &object.String{Value: string(unicode.ToUpper(r)) + s[i+len(string(r)):]}
		}
		return &object.String{Value: s}
	},
}

func init() {
	for name := range moreStringMethods {
		stringMethods = append(stringMethods, name)
	}
}

// pad adds fill (a space by default) to the start or end of s until it is
// width characters long.
func pad(name, s string, args []object.Object, atStart bool) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return wrongArgs(0, 0, "string."+name+"()", len(args), "1 or 2")
	}
	width, ok := args[0].(*object.Integer)
	if !ok {
		return wrongType(0, 0, "width for string."+name+"()", "an integer", args[0])
	}
	if width.Value > maxStringLength {
		return &object.Error{Message: strfmt.Sprintf("string.%s() would make a string too long", name)}
	}
	fill := " "
	if len(args) == 2 {
		f, ok := args[1].(*object.String)
		if !ok || f.Value == "" {
			return &object.Error{Message: strfmt.Sprintf("fill for string.%s() must be a non-empty string", name)}
		}
		fill = f.Value
	}

	missing := int(width.Value) - len([]rune(s))
	if missing <= 0 {
		return &object.String{Value: s}
	}
	padding := []rune(strings.Repeat(fill, missing/len([]rune(fill))+1))[:missing]
	if atStart {
		return &object.String{Value: string(padding) + s}
	}
	return &object.String{Value: s + string(padding)}
}

func noArgs(name string, args []object.Object) *object.Error {
	if len(args) != 0 {
		return wrongArgs(0, 0, "string."+name+"()", len(args), "0")
	}
	return nil
}

func stringArg(name string, args []object.Object) (string, *object.Error) {
	if len(args) != 1 {
		return "", wrongArgs(0, 0, "string."+name+"()", len(args), "1")
	}
	s, ok := args[0].(*object.String)
	if !ok {
		return "", wrongType(0, 0, "argument to string."+name+"()", "a string", args[0])
	}
	return s.Value, nil
}

func intArgs(name string, args []object.Object, minCount, maxCount int) ([]int64, *object.Error) {
	if len(args) < minCount || len(args) > maxCount {
		want := strconv.Itoa(minCount)
		if maxCount != minCount {
			want = strfmt.Sprintf("%d or %d", minCount, maxCount)
		}
		return nil, wrongArgs(0, 0, "string."+name+"()", len(args), want)
	}
	values := make([]int64, len(args))
	for i, arg := range args {
		n, ok := arg.(*object.Integer)
		if !ok {
			return nil, wrongType(0, 0, "arguments to string."+name+"()", "integers", arg)
		}
		values[i] = n.Value
	}
	return values, nil
}
