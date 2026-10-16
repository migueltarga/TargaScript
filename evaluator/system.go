package evaluator

import (
	"fmt"

	"github.com/migueltarga/TargaScript/object"
)

// System gives programs access to files, the environment and the exit
// code. Only the command line sets one; in the browser these builtins fail.
type System interface {
	ReadFile(path string) (string, error)
	WriteFile(path, content string, appendToFile bool) error
	ListDir(path string) ([]string, error)
	Env(name string) (string, bool)
	Exit(code int)
}

var system System

// LineReader is a System that can read the lines a user types, for input.
type LineReader interface {
	// ReadLine returns the next line without its line break, or false at
	// the end of the input.
	ReadLine() (string, bool)
}

// SetSystem sets what readFile, writeFile, listDir, env and exit use.
func SetSystem(s System) {
	system = s
}

func init() {
	builtins["readFile"] = &object.Builtin{Name: "readFile", Fn: readFileBuiltin}
	builtins["writeFile"] = &object.Builtin{Name: "writeFile", Fn: func(args ...object.Object) object.Object {
		return writeFileBuiltin("writeFile", false, args)
	}}
	builtins["appendFile"] = &object.Builtin{Name: "appendFile", Fn: func(args ...object.Object) object.Object {
		return writeFileBuiltin("appendFile", true, args)
	}}
	builtins["listDir"] = &object.Builtin{Name: "listDir", Fn: listDirBuiltin}
	builtins["env"] = &object.Builtin{Name: "env", Fn: envBuiltin}
	builtins["exit"] = &object.Builtin{Name: "exit", Fn: exitBuiltin}
	builtins["input"] = &object.Builtin{Name: "input", Fn: inputBuiltin}
}

func noSystem(name string) *object.Error {
	return &object.Error{Message: name + " is only available in the command line"}
}

func stringArgs(name string, args []object.Object, count int) ([]string, *object.Error) {
	if len(args) != count {
		return nil, &object.Error{Message: fmt.Sprintf("wrong number of arguments for %s. got=%d, want=%d", name, len(args), count)}
	}
	values := make([]string, count)
	for i, arg := range args {
		s, ok := arg.(*object.String)
		if !ok {
			return nil, &object.Error{Message: fmt.Sprintf("arguments to %s must be strings, got %s", name, typeName(arg))}
		}
		values[i] = s.Value
	}
	return values, nil
}

// readFile returns the content of a file, with a path relative to the
// working directory.
func readFileBuiltin(args ...object.Object) object.Object {
	values, err := stringArgs("readFile", args, 1)
	if err != nil {
		return err
	}
	if system == nil {
		return noSystem("readFile")
	}
	content, readErr := system.ReadFile(values[0])
	if readErr != nil {
		return &object.Error{Message: fmt.Sprintf("cannot read %q: %s", values[0], readErr)}
	}
	return &object.String{Value: content}
}

// writeFile replaces the content of a file, creating it if needed;
// appendFile adds to its end.
func writeFileBuiltin(name string, appendToFile bool, args []object.Object) object.Object {
	values, err := stringArgs(name, args, 2)
	if err != nil {
		return err
	}
	if system == nil {
		return noSystem(name)
	}
	if writeErr := system.WriteFile(values[0], values[1], appendToFile); writeErr != nil {
		return &object.Error{Message: fmt.Sprintf("cannot write %q: %s", values[0], writeErr)}
	}
	return NULL
}

// listDir returns the names in a directory, sorted.
func listDirBuiltin(args ...object.Object) object.Object {
	values, err := stringArgs("listDir", args, 1)
	if err != nil {
		return err
	}
	if system == nil {
		return noSystem("listDir")
	}
	names, listErr := system.ListDir(values[0])
	if listErr != nil {
		return &object.Error{Message: fmt.Sprintf("cannot list %q: %s", values[0], listErr)}
	}
	list := &object.Array{Elements: []object.Object{}}
	for _, name := range names {
		list.Elements = append(list.Elements, &object.String{Value: name})
	}
	return list
}

// env returns an environment variable, or the default (null if not given)
// when it isn't set.
func envBuiltin(args ...object.Object) object.Object {
	if len(args) != 1 && len(args) != 2 {
		return &object.Error{Message: fmt.Sprintf("wrong number of arguments for env. got=%d, want=1 or 2", len(args))}
	}
	name, ok := args[0].(*object.String)
	if !ok {
		return &object.Error{Message: fmt.Sprintf("name for env must be a string, got %s", typeName(args[0]))}
	}
	if system == nil {
		return noSystem("env")
	}
	if value, ok := system.Env(name.Value); ok {
		return &object.String{Value: value}
	}
	if len(args) == 2 {
		return args[1]
	}
	return NULL
}

// exit ends the program with an exit code, 0 by default.
func exitBuiltin(args ...object.Object) object.Object {
	code := int64(0)
	if len(args) > 1 {
		return &object.Error{Message: fmt.Sprintf("wrong number of arguments for exit. got=%d, want=0 or 1", len(args))}
	}
	if len(args) == 1 {
		n, ok := args[0].(*object.Integer)
		if !ok {
			return &object.Error{Message: fmt.Sprintf("exit code must be an integer, got %s", typeName(args[0]))}
		}
		code = n.Value
	}
	if system == nil {
		return noSystem("exit")
	}
	system.Exit(int(code))
	return NULL
}

// inputBuiltin prints the prompt, if any, and returns the line the user
// types, or null when the input has ended.
func inputBuiltin(args ...object.Object) object.Object {
	if len(args) > 1 {
		return &object.Error{Message: fmt.Sprintf("wrong number of arguments for input. got=%d, want=0 or 1", len(args))}
	}
	reader, ok := system.(LineReader)
	if !ok {
		return noSystem("input")
	}
	if len(args) == 1 {
		fmt.Fprint(output, objectToString(args[0]))
	}
	line, ok := reader.ReadLine()
	if !ok {
		return NULL
	}
	return &object.String{Value: line}
}
