package evaluator

import (
	"fmt"

	"github.com/migueltarga/TargaScript/object"
)

func init() {
	builtins["sleep"] = &object.Builtin{Name: "sleep", Fn: sleep}
}

// sleep returns a promise that resolves after the given number of
// milliseconds, for `await sleep(500)`.
func sleep(args ...object.Object) object.Object {
	if len(args) != 1 {
		return &object.Error{Message: fmt.Sprintf("wrong number of arguments for sleep. got=%d, want=1", len(args))}
	}

	var milliseconds int64
	switch arg := args[0].(type) {
	case *object.Integer:
		milliseconds = arg.Value
	case *object.Float:
		milliseconds = int64(arg.Value)
	default:
		return &object.Error{Message: fmt.Sprintf("argument to sleep must be a number of milliseconds, got %s", typeName(args[0]))}
	}
	if milliseconds < 0 {
		milliseconds = 0
	}
	if host == nil {
		return &object.Error{Message: "sleep is not available here"}
	}

	promise := object.NewPromise()
	host.After(milliseconds, func() { promise.Resolve(NULL) })
	return promise
}
