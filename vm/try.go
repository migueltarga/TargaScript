package vm

import (
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/object"
)

// catch looks for the innermost try block, in the frames from base up, to
// handle err. If there is one, it unwinds to it and continues at its catch
// block with the error as a value on the stack.
func (vm *VM) catch(base int, err *object.Error) bool {
	for i := len(vm.frames) - 1; i >= base; i-- {
		f := vm.frames[i]
		if len(f.handlers) == 0 {
			continue
		}
		if err.Line == 0 {
			top := vm.frames[len(vm.frames)-1]
			err.Line, err.Column = top.cl.Fn.PositionAt(top.ip)
		}

		h := f.handlers[len(f.handlers)-1]
		f.handlers = f.handlers[:len(f.handlers)-1]
		vm.frames = vm.frames[:i+1]
		vm.sp = f.base + f.cl.Fn.NumRegisters
		vm.stack[f.base+h.reg] = errorValue(err)
		f.ip = h.catch
		return true
	}
	return false
}

// errorValue is the object a catch block gets for an error.
func errorValue(err *object.Error) *object.Hash {
	hash := object.NewHash()
	hash.Set(&object.String{Value: "message"}, &object.String{Value: err.Message})
	hash.Set(&object.String{Value: "line"}, &object.Integer{Value: int64(err.Line)})
	hash.Set(&object.String{Value: "column"}, &object.Integer{Value: int64(err.Column)})
	return hash
}

// thrown is the error raised by `throw value`. Throwing a caught error again
// keeps its message and position.
func (vm *VM) thrown(f *frame, ip int, value object.Object) *object.Error {
	line, column := f.cl.Fn.PositionAt(ip)
	if hash, ok := value.(*object.Hash); ok {
		if message, ok := field(hash, "message").(*object.String); ok {
			if l, ok := field(hash, "line").(*object.Integer); ok && l.Value > 0 {
				if c, ok := field(hash, "column").(*object.Integer); ok {
					line, column = int(l.Value), int(c.Value)
				}
			}
			return evaluator.NewError(line, column, "%s", message.Value)
		}
	}
	return evaluator.NewError(line, column, "%s", evaluator.ToDisplayString(value))
}

func field(hash *object.Hash, name string) object.Object {
	pair, ok := hash.Get((&object.String{Value: name}).HashKey())
	if !ok {
		return nil
	}
	return pair.Value
}
