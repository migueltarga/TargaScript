package evaluator

import (
	"io"
	"strconv"

	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/strfmt"
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

// FileManager is a System that can also look for, delete and make files
// and directories.
type FileManager interface {
	Exists(path string) bool
	DeleteFile(path string) error
	MakeDir(path string) error
}

// CommandRunner is a System that can run other programs, for exec.
type CommandRunner interface {
	// RunCommand runs a command and returns what it wrote to its output and
	// its errors, and its exit code; err is set when it couldn't start.
	RunCommand(command Command) (stdout, stderr string, code int, err error)
}

// Command is a program for exec to run.
type Command struct {
	// Program runs with Args, or with Shell set, is a command line for the
	// system's shell
	Program string
	Args    []string
	Shell   bool

	Dir   string  // the working directory, or the script's when empty
	Input *string // what the program reads, or the script's input when nil
	Show  bool    // the output goes to the terminal as it comes instead
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
	builtins["exec"] = &object.Builtin{Name: "exec", Fn: execBuiltin}
	builtins["exists"] = &object.Builtin{Name: "exists", Fn: existsBuiltin}
	builtins["deleteFile"] = &object.Builtin{Name: "deleteFile", Fn: func(args ...object.Object) object.Object {
		return fileChange("deleteFile", "delete", FileManager.DeleteFile, args)
	}}
	builtins["makeDir"] = &object.Builtin{Name: "makeDir", Fn: func(args ...object.Object) object.Object {
		return fileChange("makeDir", "make", FileManager.MakeDir, args)
	}}
}

// existsBuiltin reports whether a file or directory exists.
func existsBuiltin(args ...object.Object) object.Object {
	values, err := stringArgs("exists", args, 1)
	if err != nil {
		return err
	}
	files, ok := system.(FileManager)
	if !ok {
		return noSystem("exists")
	}
	return nativeBoolToBooleanObject(files.Exists(values[0]))
}

// fileChange runs deleteFile, which deletes a file or an empty directory,
// or makeDir, which makes a directory and the ones it is in.
func fileChange(name, verb string, change func(FileManager, string) error, args []object.Object) object.Object {
	values, err := stringArgs(name, args, 1)
	if err != nil {
		return err
	}
	files, ok := system.(FileManager)
	if !ok {
		return noSystem(name)
	}
	if changeErr := change(files, values[0]); changeErr != nil {
		return &object.Error{Message: strfmt.Sprintf("cannot %s %q: %s", verb, values[0], changeErr)}
	}
	return NULL
}

// execBuiltin runs a command line in the shell, exec("ls -l"), or a
// program with a list of arguments, exec("git", ["log", "-1"]), and returns
// {output, errors, code, ok}. Options come last: {dir, input, show}. A
// program that fails gives its code; only one that can't start is an error.
func execBuiltin(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 3 {
		return wrongArgs(0, 0, "exec", len(args), "1 to 3")
	}
	program, ok := args[0].(*object.String)
	if !ok {
		return wrongType(0, 0, "first argument to `exec`", "a string", args[0])
	}
	command := Command{Program: program.Value, Shell: true}
	rest := args[1:]
	if len(rest) > 0 {
		if list, ok := rest[0].(*object.Array); ok {
			command.Shell = false
			command.Args = make([]string, len(list.Elements))
			for i, arg := range list.Elements {
				command.Args[i] = objectToString(arg)
			}
			rest = rest[1:]
		}
	}
	if len(rest) > 0 {
		options, ok := rest[0].(*object.Hash)
		if !ok || len(rest) > 1 {
			return &object.Error{Message: "exec takes a command, then a list of arguments or options like {\"dir\": \"src\"}, got " + typeName(rest[0])}
		}
		if err := execOptions(options, &command); err != nil {
			return err
		}
	}
	runner, ok := system.(CommandRunner)
	if !ok {
		return noSystem("exec")
	}

	stdout, stderr, code, err := runner.RunCommand(command)
	if err != nil {
		return &object.Error{Message: strfmt.Sprintf("cannot run %q: %s", program.Value, err)}
	}
	result := object.NewHash()
	result.Set(&object.String{Value: "output"}, &object.String{Value: stdout})
	result.Set(&object.String{Value: "errors"}, &object.String{Value: stderr})
	result.Set(&object.String{Value: "code"}, &object.Integer{Value: int64(code)})
	result.Set(&object.String{Value: "ok"}, nativeBoolToBooleanObject(code == 0))
	return result
}

// execOptions reads the options of exec into a command.
func execOptions(options *object.Hash, command *Command) *object.Error {
	for _, pair := range options.OrderedPairs() {
		name := objectToString(pair.Key)
		switch name {
		case "dir", "input":
			text, ok := pair.Value.(*object.String)
			if !ok {
				return wrongType(0, 0, "the "+name+" option of exec", "a string", pair.Value)
			}
			if name == "dir" {
				command.Dir = text.Value
			} else {
				command.Input = &text.Value
			}
		case "show":
			command.Show = IsTruthy(pair.Value)
		default:
			return &object.Error{Message: strfmt.Sprintf("exec has no option %q; it has dir, input and show", name)}
		}
	}
	return nil
}

func noSystem(name string) *object.Error {
	return &object.Error{Message: name + " is only available in the command line"}
}

func stringArgs(name string, args []object.Object, count int) ([]string, *object.Error) {
	if len(args) != count {
		return nil, wrongArgs(0, 0, name, len(args), strconv.Itoa(count))
	}
	values := make([]string, count)
	for i, arg := range args {
		s, ok := arg.(*object.String)
		if !ok {
			return nil, wrongType(0, 0, "arguments to "+name, "strings", arg)
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
		return &object.Error{Message: strfmt.Sprintf("cannot read %q: %s", values[0], readErr)}
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
		return &object.Error{Message: strfmt.Sprintf("cannot write %q: %s", values[0], writeErr)}
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
		return &object.Error{Message: strfmt.Sprintf("cannot list %q: %s", values[0], listErr)}
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
		return wrongArgs(0, 0, "env", len(args), "1 or 2")
	}
	name, ok := args[0].(*object.String)
	if !ok {
		return wrongType(0, 0, "name for env", "a string", args[0])
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
		return wrongArgs(0, 0, "exit", len(args), "0 or 1")
	}
	if len(args) == 1 {
		n, ok := args[0].(*object.Integer)
		if !ok {
			return wrongType(0, 0, "exit code", "an integer", args[0])
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
		return wrongArgs(0, 0, "input", len(args), "0 or 1")
	}
	reader, ok := system.(LineReader)
	if !ok {
		return noSystem("input")
	}
	if len(args) == 1 {
		io.WriteString(output, objectToString(args[0]))
	}
	line, ok := reader.ReadLine()
	if !ok {
		return NULL
	}
	return &object.String{Value: line}
}
