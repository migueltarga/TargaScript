package evaluator

import (
	"fmt"

	"github.com/migueltarga/TargaScript/object"
)

// builtins is a map of built-in functions that can be called from TargaScript
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
}
