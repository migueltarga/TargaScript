//go:build tinygo && wasm

// Package jsbridge lets TargaScript code use JavaScript values when the
// interpreter runs in a browser or in Node, through the functions that
// targascript.js provides.
package jsbridge

import (
	"math"

	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/object"
)

const JS_OBJ = "JS"

// Value is a JavaScript object or function seen from TargaScript.
type Value struct {
	handle uint32
}

var (
	// Values already wrapped, by handle, so the same JavaScript object is
	// always the same TargaScript value and == works on it.
	wrapped = map[uint32]*Value{}

	// TargaScript functions passed to JavaScript, by callback id, and the
	// handle of the JavaScript function made for each.
	callbacks       []object.Object
	callbackHandles = map[object.Object]uint32{}

	// Callback ids freed by one-shot functions, to reuse
	freeCallbacks []uint32
)

// Global returns globalThis, which scripts see as `js`.
func Global() *Value {
	return wrap(handleGlobal)
}

func wrap(handle uint32) *Value {
	if v, ok := wrapped[handle]; ok {
		hostRelease(handle)
		return v
	}
	v := &Value{handle: handle}
	wrapped[handle] = v
	return v
}

func (v *Value) Type() object.ObjectType { return JS_OBJ }

func (v *Value) Inspect() string {
	ptr, n := stringPtr("String")
	stringFn := hostGet(handleGlobal, ptr, n)
	result := hostInvoke(stringFn, &v.handle, 1)
	hostRelease(stringFn)
	if result&failed != 0 {
		hostRelease(result &^ failed)
		return "<js>"
	}
	text := readString(result)
	hostRelease(result)
	return text
}

func (v *Value) Get(name string) object.Object {
	ptr, n := stringPtr(name)
	return fromResult(hostGet(v.handle, ptr, n))
}

func (v *Value) Set(name string, value object.Object) object.Object {
	ptr, n := stringPtr(name)
	h, temporary := toHost(value)
	result := hostSet(v.handle, ptr, n, h)
	if temporary {
		hostRelease(h)
	}
	if err := failure(result); err != nil {
		return err
	}
	return value
}

func (v *Value) Call(args []object.Object) object.Object {
	handles, release := toHostArgs(args)
	defer release()
	ptr, n := argsPtr(handles)
	return fromResult(hostInvoke(v.handle, ptr, n))
}

func (v *Value) CallMethod(name string, args []object.Object) object.Object {
	handles, release := toHostArgs(args)
	defer release()
	namePtr, nameLen := stringPtr(name)
	ptr, n := argsPtr(handles)
	return fromResult(hostCall(v.handle, namePtr, nameLen, ptr, n))
}

func (v *Value) New(args []object.Object) object.Object {
	handles, release := toHostArgs(args)
	defer release()
	ptr, n := argsPtr(handles)
	return fromResult(hostConstruct(v.handle, ptr, n))
}

// failure returns the error for a handle returned with the failed flag, or
// nil.
func failure(result uint32) *object.Error {
	if result&failed == 0 {
		return nil
	}
	message := readString(result &^ failed)
	hostRelease(result &^ failed)
	return &object.Error{Message: message}
}

func fromResult(result uint32) object.Object {
	if err := failure(result); err != nil {
		return err
	}
	return fromHost(result)
}

// fromHost converts a JavaScript value to TargaScript, taking over the
// reference held by handle. Numbers with no fractional part become
// integers, and arrays are copied into TargaScript arrays. Other objects and
// functions are wrapped in a Value.
func fromHost(handle uint32) object.Object {
	switch hostKind(handle) {
	case kindUndefined, kindNull:
		return evaluator.NULL
	case kindBoolean:
		if handle == handleTrue {
			return evaluator.TRUE
		}
		return evaluator.FALSE
	case kindNumber:
		f := hostNumber(handle)
		hostRelease(handle)
		if f == math.Trunc(f) && math.Abs(f) <= 1<<53 {
			return &object.Integer{Value: int64(f)}
		}
		return &object.Float{Value: f}
	case kindString:
		text := readString(handle)
		hostRelease(handle)
		return &object.String{Value: text}
	case kindArray:
		elements := make([]object.Object, hostLength(handle))
		for i := range elements {
			elements[i] = fromHost(hostIndex(handle, uint32(i)))
		}
		hostRelease(handle)
		return &object.Array{Elements: elements}
	default:
		return wrap(handle)
	}
}

// toHost converts a TargaScript value to JavaScript. When temporary is true
// the caller must release the handle once JavaScript is done with it.
// Arrays and hashes are copied, and functions become JavaScript functions
// that call back into the script.
func toHost(obj object.Object) (handle uint32, temporary bool) {
	switch obj := obj.(type) {
	case *Value:
		return obj.handle, false
	case *object.Integer:
		return hostMakeNumber(float64(obj.Value)), true
	case *object.Float:
		return hostMakeNumber(obj.Value), true
	case *object.String:
		ptr, n := stringPtr(obj.Value)
		return hostMakeString(ptr, n), true
	case *object.Boolean:
		if obj.Value {
			return handleTrue, false
		}
		return handleFalse, false
	case *object.Array:
		array := hostMakeArray()
		for _, element := range obj.Elements {
			h, temporary := toHost(element)
			hostPush(array, h)
			if temporary {
				hostRelease(h)
			}
		}
		return array, true
	case *object.Hash:
		hash := hostMakeObject()
		for _, pair := range obj.OrderedPairs() {
			key := pair.Key.Inspect()
			if s, ok := pair.Key.(*object.String); ok {
				key = s.Value
			}
			ptr, n := stringPtr(key)
			h, temporary := toHost(pair.Value)
			hostSet(hash, ptr, n, h)
			if temporary {
				hostRelease(h)
			}
		}
		return hash, true
	case *object.Builtin, *object.Closure:
		if h, ok := callbackHandles[obj]; ok {
			return h, false
		}
		h := hostMakeFunction(registerCallback(obj))
		callbackHandles[obj] = h
		return h, false
	case *object.Promise:
		return toJSPromise(obj), true
	default:
		return handleUndefined, false
	}
}

func toHostArgs(args []object.Object) ([]uint32, func()) {
	handles := make([]uint32, len(args))
	var temporaries []uint32
	for i, arg := range args {
		h, temporary := toHost(arg)
		handles[i] = h
		if temporary {
			temporaries = append(temporaries, h)
		}
	}
	return handles, func() {
		for _, h := range temporaries {
			hostRelease(h)
		}
	}
}

// Callback runs the TargaScript function with the given callback id for
// JavaScript, with the arguments in the array args. It returns the handle
// of the result, which JavaScript releases.
func Callback(id uint32, args uint32) uint32 {
	converted, ok := fromHost(args).(*object.Array)
	if !ok || int(id) >= len(callbacks) || callbacks[id] == nil {
		return handleUndefined
	}

	result := evaluator.CallFunction(callbacks[id], converted.Elements)
	if err, ok := result.(*object.Error); ok {
		ConsoleError(ErrorMessage(err))
		return handleUndefined
	}

	return Return(result)
}

// Keep wraps a JavaScript value passed to Go as a handle, keeping it after
// JavaScript releases the handle.
func Keep(handle uint32) *Value {
	hostRetain(handle)
	return wrap(handle)
}

// Return converts a value returned to JavaScript to a handle that
// JavaScript releases.
func Return(obj object.Object) uint32 {
	h, temporary := toHost(obj)
	if !temporary {
		hostRetain(h)
	}
	return h
}

// ReadString returns the JavaScript string with the given handle.
func ReadString(handle uint32) string {
	return readString(handle)
}

// ConsoleError writes message with console.error.
func ConsoleError(message string) {
	console("error", message)
}

// ConsoleLog writes message with console.log.
func ConsoleLog(message string) {
	console("log", message)
}

func console(method, message string) {
	if c, ok := Global().Get("console").(*Value); ok {
		c.CallMethod(method, []object.Object{&object.String{Value: message}})
	}
}

// ErrorMessage formats a runtime error with its position, as the CLI does.
func ErrorMessage(err *object.Error) string {
	if err.Line == 0 {
		return err.Message
	}
	return err.Message + " (line " + itoa(err.Line) + ", column " + itoa(err.Column) + ")"
}

func itoa(n int) string {
	return (&object.Integer{Value: int64(n)}).Inspect()
}

// registerCallback gives a function an id JavaScript can call it by.
func registerCallback(fn object.Object) uint32 {
	if n := len(freeCallbacks); n > 0 {
		id := freeCallbacks[n-1]
		freeCallbacks = freeCallbacks[:n-1]
		callbacks[id] = fn
		return id
	}
	callbacks = append(callbacks, fn)
	return uint32(len(callbacks) - 1)
}

// oneShotPair makes two JavaScript functions of which only the first call
// counts, like the resolve and reject functions of a promise. Once one is
// called, both are freed.
func oneShotPair(first, second func(object.Object)) (uint32, uint32) {
	var ids [2]uint32
	var handles [2]uint32
	called := false
	release := func() {
		for i := range ids {
			callbacks[ids[i]] = nil
			freeCallbacks = append(freeCallbacks, ids[i])
			hostRelease(handles[i])
		}
	}
	wrap := func(fn func(object.Object)) *object.Builtin {
		return &object.Builtin{Fn: func(args ...object.Object) object.Object {
			if called {
				return evaluator.NULL
			}
			called = true
			release()
			var arg object.Object = evaluator.NULL
			if len(args) > 0 {
				arg = args[0]
			}
			fn(arg)
			return evaluator.NULL
		}}
	}
	ids[0] = registerCallback(wrap(first))
	ids[1] = registerCallback(wrap(second))
	handles[0] = hostMakeFunction(ids[0])
	handles[1] = hostMakeFunction(ids[1])
	return handles[0], handles[1]
}

// IsAwaitable reports whether the value is a promise or another thenable.
func (v *Value) IsAwaitable() bool {
	ptr, n := stringPtr("then")
	then := hostGet(v.handle, ptr, n)
	if then&failed != 0 {
		hostRelease(then &^ failed)
		return false
	}
	isFunction := hostKind(then) == kindFunction
	hostRelease(then)
	return isFunction
}

// Then waits for the promise, as await does.
func (v *Value) Then(resolve func(object.Object), reject func(object.Object)) {
	onResolve, onReject := oneShotPair(resolve, func(reason object.Object) {
		reject(&object.Error{Message: reasonMessage(reason)})
	})
	args := []uint32{onResolve, onReject}
	ptr, n := stringPtr("then")
	result := hostCall(v.handle, ptr, n, &args[0], 2)
	if err := failure(result); err != nil {
		reject(err)
		return
	}
	hostRelease(result)
}

// reasonMessage describes why a JavaScript promise was rejected: the
// message of an Error, or the reason as text.
func reasonMessage(reason object.Object) string {
	if v, ok := reason.(*Value); ok {
		if message, ok := v.Get("message").(*object.String); ok {
			return message.Value
		}
	}
	return evaluator.ToDisplayString(reason)
}

// toJSPromise returns a JavaScript promise that settles with p.
func toJSPromise(p *object.Promise) uint32 {
	executor := &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 2 {
			return evaluator.NULL
		}
		resolve, reject := args[0], args[1]
		p.Then(
			func(value object.Object) { evaluator.CallFunction(resolve, []object.Object{value}) },
			func(reason object.Object) {
				message := reasonMessage(reason)
				if err, ok := reason.(*object.Error); ok {
					message = ErrorMessage(err)
				}
				errorType, _ := Global().Get("Error").(*Value)
				evaluator.CallFunction(reject, []object.Object{errorType.New([]object.Object{&object.String{Value: message}})})
			},
		)
		return evaluator.NULL
	}}

	// The Promise constructor calls the executor right away, so it can be
	// freed as soon as the promise exists
	id := registerCallback(executor)
	executorHandle := hostMakeFunction(id)
	ptr, n := stringPtr("Promise")
	promiseType := hostGet(handleGlobal, ptr, n)
	result := hostConstruct(promiseType, &executorHandle, 1)
	hostRelease(promiseType)
	hostRelease(executorHandle)
	callbacks[id] = nil
	freeCallbacks = append(freeCallbacks, id)
	if result&failed != 0 {
		hostRelease(result &^ failed)
		return handleUndefined
	}
	return result
}
