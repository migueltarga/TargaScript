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

func TestStackDepths(t *testing.T) {
	// if c { 1 } else { 2 }: both ways reach the end with one value
	ins := instructions{
		op(code.OpTrue),             // 0
		op(code.OpJumpNotTruthy, 4), // 1
		op(code.OpConstant, 0),      // 2
		op(code.OpJump, 5),          // 3
		op(code.OpConstant, 1),      // 4
		op(code.OpReturnValue),      // 5
	}
	depths, err := stackDepths(ins)
	if err != nil {
		t.Fatal(err)
	}
	for at, want := range map[int]int{0: 0, 1: 1, 2: 0, 3: 1, 4: 0, 5: 1} {
		if depths[at] != want {
			t.Errorf("depth at %d is %d, want %d", at, depths[at], want)
		}
	}

	// A way that leaves a value more is a bug of the compiler
	bad := instructions{
		op(code.OpTrue),             // 0
		op(code.OpJumpNotTruthy, 5), // 1
		op(code.OpNull),             // 2
		op(code.OpNull),             // 3
		op(code.OpNull),             // 4
		op(code.OpReturnValue),      // 5
	}
	if _, err := stackDepths(bad); err == nil || !strings.Contains(err.Error(), "different stack depths") {
		t.Errorf("got %v, want an error about different depths", err)
	}
}
