package compiler

import (
	"bytes"
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/code"
)

func TestStackDepths(t *testing.T) {
	join := func(parts ...[]byte) code.Instructions { return bytes.Join(parts, nil) }

	// if c { 1 } else { 2 }: both ways reach the end with one value
	ins := join(
		code.Make(code.OpTrue),              // 0
		code.Make(code.OpJumpNotTruthy, 10), // 1
		code.Make(code.OpConstant, 0),       // 4
		code.Make(code.OpJump, 13),          // 7
		code.Make(code.OpConstant, 1),       // 10
		code.Make(code.OpReturnValue),       // 13
	)
	depths, err := stackDepths(ins)
	if err != nil {
		t.Fatal(err)
	}
	for at, want := range map[int]int{0: 0, 1: 1, 4: 0, 7: 1, 10: 0, 13: 1} {
		if depths[at] != want {
			t.Errorf("depth at %d is %d, want %d", at, depths[at], want)
		}
	}

	// A way that leaves a value more is a bug of the compiler
	bad := join(
		code.Make(code.OpTrue),             // 0
		code.Make(code.OpJumpNotTruthy, 7), // 1
		code.Make(code.OpNull),             // 4
		code.Make(code.OpNull),             // 5
		code.Make(code.OpNull),             // 6
		code.Make(code.OpReturnValue),      // 7
	)
	if _, err := stackDepths(bad); err == nil || !strings.Contains(err.Error(), "reached with") {
		t.Errorf("got %v, want an error about different depths", err)
	}
}
