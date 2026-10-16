package vm

import (
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/object"
)

// A program that awaits a promise pauses: its frames and stack are saved in
// a fiber and the VM returns to its caller, normally JavaScript's event
// loop. When the promise settles, the fiber is put back and resumes.

// fiber is a paused run.
type fiber struct {
	frames []*frame
	stack  []object.Object

	// promise settles with the run's result once it finishes
	promise *object.Promise

	// Position of the await, for the error when the promise is rejected
	line, column int
}

// suspension is what loop returns to pause at an await.
type suspension struct {
	awaited      object.Awaitable
	line, column int
}

func (s *suspension) Type() object.ObjectType { return "SUSPENSION" }
func (s *suspension) Inspect() string         { return "suspension" }

// suspend saves the frames from base up, and everything they have on the
// stack, into fb. It then waits for the awaited promise to resume it.
func (vm *VM) suspend(base int, fb *fiber, s *suspension) {
	bottom := vm.frames[base].base - 1

	fb.stack = append([]object.Object{}, vm.stack[bottom:vm.sp]...)
	fb.frames = append([]*frame{}, vm.frames[base:]...)
	for _, f := range fb.frames {
		f.base -= bottom
		for i := range f.loops {
			f.loops[i].sp -= bottom
		}
		for i := range f.handlers {
			f.handlers[i].sp -= bottom
		}
	}
	fb.line, fb.column = s.line, s.column

	for i := bottom; i < vm.sp; i++ {
		vm.stack[i] = nil
	}
	vm.sp = bottom
	vm.frames = vm.frames[:base]

	s.awaited.Then(
		func(value object.Object) { vm.resume(fb, value, nil) },
		func(reason object.Object) { vm.resume(fb, nil, reason) },
	)
}

// resume continues a paused run with the value its promise resolved to, or
// fails it at the await with the reason the promise was rejected.
func (vm *VM) resume(fb *fiber, value, reason object.Object) {
	start := vm.sp
	base := len(vm.frames)

	vm.ensureStack(start + len(fb.stack) + 256)
	copy(vm.stack[start:], fb.stack)
	vm.sp = start + len(fb.stack)
	for _, f := range fb.frames {
		f.base += start
		for i := range f.loops {
			f.loops[i].sp += start
		}
		for i := range f.handlers {
			f.handlers[i].sp += start
		}
	}
	vm.frames = append(vm.frames, fb.frames...)
	fb.frames, fb.stack = nil, nil

	if reason != nil {
		vm.pendingError = evaluator.NewError(fb.line, fb.column, "%s", rejectionMessage(reason))
	} else {
		vm.push(value)
	}

	vm.asyncBases = append(vm.asyncBases, base)
	result := vm.runFiber(base, fb)
	vm.asyncBases = vm.asyncBases[:len(vm.asyncBases)-1]
	if result == fb.promise {
		return // paused again
	}
	if err, ok := result.(*object.Error); ok {
		fb.promise.Reject(err)
		return
	}
	fb.promise.Resolve(result)
}

// rejectionMessage describes why a promise was rejected.
func rejectionMessage(reason object.Object) string {
	switch reason := reason.(type) {
	case *object.Error:
		return reason.Message
	case nil:
		return "promise rejected"
	default:
		return evaluator.ToDisplayString(reason)
	}
}
