//go:build tinygo && wasm

package jsbridge

import (
	"errors"

	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/object"
)

// Regex runs string patterns with JavaScript's RegExp, so Go's regexp
// package stays out of the WebAssembly build.
type Regex struct{}

func (Regex) Compile(pattern string) (evaluator.Regex, error) {
	re := jsRegex{pattern: &object.String{Value: pattern}}
	if _, err := re.regexp(""); err != nil {
		return nil, err
	}
	return re, nil
}

type jsRegex struct{ pattern *object.String }

// regexp makes a RegExp for the pattern with the given flags.
func (r jsRegex) regexp(flags string) (*Value, error) {
	result := Global().Get("RegExp").(*Value).New([]object.Object{r.pattern, &object.String{Value: flags}})
	if err, ok := result.(*object.Error); ok {
		return nil, errors.New(err.Message)
	}
	return result.(*Value), nil
}

// stringMethod calls a method of JavaScript strings on s.
func stringMethod(name, s string, args ...object.Object) object.Object {
	method := Global().Get("String").(*Value).Get("prototype").(*Value).Get(name).(*Value)
	return method.CallMethod("call", append([]object.Object{&object.String{Value: s}}, args...))
}

func strings(obj object.Object) []string {
	array, ok := obj.(*object.Array)
	if !ok {
		return nil
	}
	values := make([]string, len(array.Elements))
	for i, element := range array.Elements {
		// Groups that didn't take part in the match are undefined
		if s, ok := element.(*object.String); ok {
			values[i] = s.Value
		}
	}
	return values
}

func (r jsRegex) MatchString(s string) bool {
	re, _ := r.regexp("")
	result, _ := re.CallMethod("test", []object.Object{&object.String{Value: s}}).(*object.Boolean)
	return result != nil && result.Value
}

func (r jsRegex) FindStringSubmatch(s string) []string {
	re, _ := r.regexp("")
	return strings(re.CallMethod("exec", []object.Object{&object.String{Value: s}}))
}

func (r jsRegex) FindAllStringSubmatch(s string) [][]string {
	re, _ := r.regexp("g")
	matches := Global().Get("Array").(*Value).CallMethod("from", []object.Object{stringMethod("matchAll", s, re)})
	all := [][]string{}
	if array, ok := matches.(*object.Array); ok {
		for _, m := range array.Elements {
			all = append(all, strings(m))
		}
	}
	return all
}

func (r jsRegex) ReplaceAllString(s, replacement string) string {
	re, _ := r.regexp("g")
	result, _ := stringMethod("replace", s, re, &object.String{Value: replacement}).(*object.String)
	if result == nil {
		return s
	}
	return result.Value
}

func (r jsRegex) Split(s string) []string {
	re, _ := r.regexp("")
	return strings(stringMethod("split", s, re))
}

func (r jsRegex) ReplaceAllFunc(s string, replace func([]string) string) string {
	re, _ := r.regexp("g")
	// JavaScript calls the function with the match, each group, then the
	// position and the string; the number of groups comes from matching
	// the pattern, made to match "", against ""
	empty, _ := (jsRegex{&object.String{Value: r.pattern.Value + "|"}}).regexp("")
	groups := len(strings(empty.CallMethod("exec", []object.Object{&object.String{Value: ""}})))
	fn := &object.Builtin{Name: "replace", Fn: func(args ...object.Object) object.Object {
		values := make([]string, groups)
		for i := 0; i < groups && i < len(args); i++ {
			if str, ok := args[i].(*object.String); ok {
				values[i] = str.Value
			}
		}
		return &object.String{Value: replace(values)}
	}}
	// The function is only needed during this call, so it's freed after
	id := registerCallback(fn)
	handle := hostMakeFunction(id)
	result, _ := stringMethod("replace", s, re, &Value{handle: handle}).(*object.String)
	callbacks[id] = nil
	freeCallbacks = append(freeCallbacks, id)
	hostRelease(handle)
	if result == nil {
		return s
	}
	return result.Value
}
