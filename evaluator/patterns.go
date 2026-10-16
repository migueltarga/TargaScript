package evaluator

import (
	"strconv"

	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/strfmt"
)

// RegexEngine compiles regular expressions. The command line uses Go's
// regexp package and the browser JavaScript's RegExp, which keeps Go's
// regexp out of the WebAssembly build. Both share the common syntax.
type RegexEngine interface {
	Compile(pattern string) (Regex, error)
}

// Regex is a compiled regular expression.
type Regex interface {
	MatchString(s string) bool
	// FindStringSubmatch returns the match and its groups, or nil
	FindStringSubmatch(s string) []string
	FindAllStringSubmatch(s string) [][]string
	// ReplaceAllString replaces every match, with $1 for groups
	ReplaceAllString(s, replacement string) string
	// ReplaceAllFunc replaces every match with what replace returns for
	// the match and its groups; a group that didn't take part is ""
	ReplaceAllFunc(s string, replace func(groups []string) string) string
	Split(s string) []string
}

var (
	regexEngine RegexEngine
	regexCache  = map[string]Regex{}
)

// SetRegexEngine sets what the pattern methods of strings use.
func SetRegexEngine(engine RegexEngine) {
	regexEngine = engine
	regexCache = map[string]Regex{}
}

func init() {
	moreStringMethods["matches"] = func(s string, args []object.Object) object.Object {
		re, err := regexArg("matches", args, 1)
		if err != nil {
			return err
		}
		return nativeBoolToBooleanObject(re.MatchString(s))
	}
	moreStringMethods["match"] = func(s string, args []object.Object) object.Object {
		re, err := regexArg("match", args, 1)
		if err != nil {
			return err
		}
		found := re.FindStringSubmatch(s)
		if found == nil {
			return NULL
		}
		return stringArray(found)
	}
	moreStringMethods["matchAll"] = func(s string, args []object.Object) object.Object {
		re, err := regexArg("matchAll", args, 1)
		if err != nil {
			return err
		}
		all := &object.Array{Elements: []object.Object{}}
		for _, found := range re.FindAllStringSubmatch(s) {
			all.Elements = append(all.Elements, stringArray(found))
		}
		return all
	}
	moreStringMethods["replacePattern"] = func(s string, args []object.Object) object.Object {
		re, err := regexArg("replacePattern", args, 2)
		if err != nil {
			return err
		}
		// A function gets the match and its groups and returns the text
		// to put in its place
		if isCallable(args[1]) {
			var failed object.Object
			result := re.ReplaceAllFunc(s, func(groups []string) string {
				if failed != nil {
					return ""
				}
				values := make([]object.Object, len(groups))
				for i, group := range groups {
					values[i] = &object.String{Value: group}
				}
				replaced := applyFunction(args[1], values)
				if isError(replaced) {
					failed = replaced
					return ""
				}
				return objectToString(replaced)
			})
			if failed != nil {
				return failed
			}
			return &object.String{Value: result}
		}
		replacement, ok := args[1].(*object.String)
		if !ok {
			return &object.Error{Message: strfmt.Sprintf("replacement for string.replacePattern() must be a string or a function, got %s", typeName(args[1]))}
		}
		return &object.String{Value: re.ReplaceAllString(s, replacement.Value)}
	}
	moreStringMethods["splitPattern"] = func(s string, args []object.Object) object.Object {
		re, err := regexArg("splitPattern", args, 1)
		if err != nil {
			return err
		}
		return stringArray(re.Split(s))
	}
	stringMethods = append(stringMethods, "matches", "match", "matchAll", "replacePattern", "splitPattern")
}

// regexArg compiles the pattern in the first argument, caching it.
func regexArg(name string, args []object.Object, count int) (Regex, *object.Error) {
	if len(args) != count {
		return nil, wrongArgs(0, 0, "string."+name+"()", len(args), strconv.Itoa(count))
	}
	pattern, ok := args[0].(*object.String)
	if !ok {
		return nil, &object.Error{Message: strfmt.Sprintf("pattern for string.%s() must be a string, got %s", name, typeName(args[0]))}
	}
	if regexEngine == nil {
		return nil, &object.Error{Message: "patterns are not available here"}
	}
	if re, ok := regexCache[pattern.Value]; ok {
		return re, nil
	}
	re, err := regexEngine.Compile(pattern.Value)
	if err != nil {
		return nil, &object.Error{Message: strfmt.Sprintf("invalid pattern %q: %s", pattern.Value, err)}
	}
	regexCache[pattern.Value] = re
	return re, nil
}

func stringArray(values []string) *object.Array {
	array := &object.Array{Elements: make([]object.Object, len(values))}
	for i, v := range values {
		array.Elements[i] = &object.String{Value: v}
	}
	return array
}
