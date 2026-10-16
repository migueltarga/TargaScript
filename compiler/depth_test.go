package compiler

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/code"
)

func op(o code.Opcode, args ...int) instruction {
	ins := instruction{op: o}
	copy(ins.args[:], args)
	return ins
}

func flowOf(ins instructions) *emitFlow {
	f := newEmitFlow()
	for at, i := range ins {
		f.add(at, i)
	}
	return f
}

func TestStackDepths(t *testing.T) {
	// if c { 1 } else { 2 }: both ways reach the end with one value
	f := flowOf(instructions{
		op(code.OpTrue),             // 0
		op(code.OpJumpNotTruthy, 4), // 1
		op(code.OpConstant, 0),      // 2
		op(code.OpJump, 5),          // 3
		op(code.OpConstant, 1),      // 4
		op(code.OpReturnValue),      // 5
	})
	if f.err != nil {
		t.Fatal(f.err)
	}
	for at, want := range map[int]int{0: 0, 1: 1, 2: 0, 3: 1, 4: 0, 5: 1} {
		if f.depths[at] != want {
			t.Errorf("depth at %d is %d, want %d", at, f.depths[at], want)
		}
	}

	// A way that leaves a value more is a bug of the compiler
	f = flowOf(instructions{
		op(code.OpTrue),             // 0
		op(code.OpJumpNotTruthy, 5), // 1
		op(code.OpNull),             // 2
		op(code.OpNull),             // 3
		op(code.OpNull),             // 4
		op(code.OpReturnValue),      // 5
	})
	if f.err == nil || !strings.Contains(f.err.Error(), "different stack depths") {
		t.Errorf("got %v, want an error about different depths", f.err)
	}
}
