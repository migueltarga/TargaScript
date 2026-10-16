//go:build vmcheck

package vm

import "fmt"

const checkDepths = true

// checkDepth panics when the stack of the running function doesn't have the
// depth the compiler worked out for the instruction at ip.
func (vm *VM) checkDepth(f *frame, ip int) {
	want := f.cl.Fn.Depths[ip]
	if got := vm.sp - f.base - f.cl.Fn.NumLocals; got != want {
		panic(fmt.Sprintf("%s at %d: stack has %d values, the compiler expected %d\n%s",
			f.cl.Fn.Name, ip, got, want, f.cl.Fn.Instructions))
	}
}
