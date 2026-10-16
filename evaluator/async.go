package evaluator

import (
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/strfmt"
)

func init() {
	builtins["sleep"] = &object.Builtin{Name: "sleep", Fn: sleep}
	builtins["all"] = &object.Builtin{Name: "all", Fn: all}
}

// sleep returns a promise that resolves after the given number of
// milliseconds, for `await sleep(500)`.
func sleep(args ...object.Object) object.Object {
	return sleepWithHost(host, args...)
}

func sleepWithHost(h Host, args ...object.Object) object.Object {
	if len(args) != 1 {
		return wrongArgs(0, 0, "sleep", len(args), "1")
	}

	var milliseconds int64
	switch arg := args[0].(type) {
	case *object.Integer:
		milliseconds = arg.Value
	case *object.Float:
		milliseconds = int64(arg.Value)
	default:
		return &object.Error{Message: strfmt.Sprintf("argument to sleep must be a number of milliseconds, got %s", typeName(args[0]))}
	}
	if milliseconds < 0 {
		milliseconds = 0
	}
	if h == nil {
		return &object.Error{Message: "sleep is not available here"}
	}

	promise := object.NewPromise()
	h.After(milliseconds, func() { promise.Resolve(NULL) })
	return promise
}

// all returns a promise of the results of an array of promises, in the same
// order, for `await all(urls.map(url => fetch(url)))`. Values that aren't
// promises are results already. It's rejected as soon as one is.
func all(args ...object.Object) object.Object {
	if len(args) != 1 {
		return wrongArgs(0, 0, "all", len(args), "1")
	}
	list, ok := args[0].(*object.Array)
	if !ok {
		return &object.Error{Message: strfmt.Sprintf("argument to all must be an array of promises, got %s", typeName(args[0]))}
	}

	promise := object.NewPromise()
	results := make([]object.Object, len(list.Elements))
	// One more than the promises still pending, so a promise that's already
	// settled, and calls back at once, can't resolve before the loop ends
	waiting := 1
	finished := func() {
		waiting--
		if waiting == 0 {
			promise.Resolve(&object.Array{Elements: results})
		}
	}
	for i, element := range list.Elements {
		awaitable, ok := element.(object.Awaitable)
		if !ok || !awaitable.IsAwaitable() {
			results[i] = element
			continue
		}
		waiting++
		awaitable.Then(func(value object.Object) {
			results[i] = value
			finished()
		}, func(reason object.Object) {
			promise.Reject(reason)
		})
	}
	finished()
	return promise
}
