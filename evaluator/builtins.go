package evaluator

import (
	"fmt"

	"github.com/migueltarga/TargaScript/object"
)

// builtins is a map of built-in functions that can be used in TargaScript
var builtins = map[string]*object.Builtin{
	"print": &object.Builtin{
		Name: "print",
		Fn: func(args ...object.Object) object.Object {
			for _, arg := range args {
				fmt.Println(arg.Inspect())
			}

			return NULL
		},
	},

	// Add a built-in range function for creating arrays of sequential integers
	"range": &object.Builtin{
		Name: "range",
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{
					Message: fmt.Sprintf("wrong number of arguments for range. got=%d, want=2", len(args)),
				}
			}

			start, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{
					Message: fmt.Sprintf("first argument to range must be INTEGER, got %s", args[0].Type()),
				}
			}

			end, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{
					Message: fmt.Sprintf("second argument to range must be INTEGER, got %s", args[1].Type()),
				}
			}

			startVal := start.Value
			endVal := end.Value

			// If start > end, return empty array
			if startVal > endVal {
				return &object.Array{Elements: []object.Object{}}
			}

			// Create the range array
			elements := make([]object.Object, 0, endVal-startVal+1)
			for i := startVal; i <= endVal; i++ {
				elements = append(elements, &object.Integer{Value: i})
			}

			return &object.Array{Elements: elements}
		},
	},
}
