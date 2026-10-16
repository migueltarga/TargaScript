package object

const EXTERNAL_OBJ = "EXTERNAL"

// External is a value that lives outside the interpreter, such as a
// JavaScript object in the browser. The evaluator forwards property access,
// calls and construction to it:
//
//	x.name         Get("name")
//	x.name = v     Set("name", v)
//	x[key]         Get(key), for string and integer keys
//	x.name(args)   CallMethod("name", args)
//	x(args)        Call(args)
//	x.new(args)    New(args)
//
// Each method returns the result or an *Error.
type External interface {
	Object
	Get(name string) Object
	Set(name string, value Object) Object
	Call(args []Object) Object
	CallMethod(name string, args []Object) Object
	New(args []Object) Object
}
