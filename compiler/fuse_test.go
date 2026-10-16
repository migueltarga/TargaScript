package compiler

import (
	"bytes"
	"testing"

	"github.com/migueltarga/TargaScript/code"
)

func TestFuse(t *testing.T) {
	join := func(parts ...[]byte) code.Instructions { return bytes.Join(parts, nil) }
	tests := []struct {
		name     string
		in, want code.Instructions
	}{
		{
			"constant then operator",
			join(code.Make(code.OpGetLocal, 0), code.Make(code.OpConstant, 1), code.Make(code.OpSub)),
			join(code.Make(code.OpGetLocal, 0), code.Make(code.OpBinaryConst, int(code.OpSub), 1)),
		},
		{
			"comparison then jump",
			join(code.Make(code.OpLessThan), code.Make(code.OpJumpNotTruthy, 4)),
			join(code.Make(code.OpCompareJump, int(code.OpLessThan), 4)),
		},
		{
			"set then pop",
			join(code.Make(code.OpSetLocal, 2), code.Make(code.OpPop), code.Make(code.OpSetGlobal, 3), code.Make(code.OpPop)),
			join(code.Make(code.OpSetLocalPop, 2, 0), code.Make(code.OpSetGlobalPop, 3, 0)),
		},
		{
			"constant, comparison and jump",
			join(code.Make(code.OpGetLocal, 0), code.Make(code.OpConstant, 1), code.Make(code.OpLessThan), code.Make(code.OpJumpNotTruthy, 9)),
			join(code.Make(code.OpGetLocal, 0), code.Make(code.OpCompareConstJump, int(code.OpLessThan), 1, 9, 0)),
		},
		{
			// x + (if c { 1 } else { 2 }): the jump to OpAdd keeps it
			"jump to the second instruction",
			join(code.Make(code.OpJump, 6), code.Make(code.OpConstant, 0), code.Make(code.OpAdd)),
			join(code.Make(code.OpJump, 6), code.Make(code.OpConstant, 0), code.Make(code.OpAdd)),
		},
	}
	for _, tt := range tests {
		fusePairs(tt.in, jumpTargets(tt.in))
		if !bytes.Equal(tt.in, tt.want) {
			t.Errorf("%s:\ngot\n%swant\n%s", tt.name, tt.in, tt.want)
		}
	}
}

func TestFuseLocals(t *testing.T) {
	join := func(parts ...[]byte) code.Instructions { return bytes.Join(parts, nil) }
	tests := []struct {
		name     string
		in, want code.Instructions
	}{
		{
			"n - 1",
			join(code.Make(code.OpGetLocal, 0), code.Make(code.OpConstant, 1), code.Make(code.OpSub)),
			join(code.Make(code.OpLocalBinaryConst, 0, int(code.OpSub), 1, 0)),
		},
		{
			"if n < 2",
			join(code.Make(code.OpGetLocal, 0), code.Make(code.OpConstant, 1), code.Make(code.OpLessThan), code.Make(code.OpJumpNotTruthy, 10)),
			join(code.Make(code.OpLocalCompareConstJump, 0, int(code.OpLessThan), 1, 10, 0)),
		},
		{
			"a + b",
			join(code.Make(code.OpGetLocal, 0), code.Make(code.OpGetLocal, 1), code.Make(code.OpAdd)),
			join(code.Make(code.OpLocalLocalBinary, 0, 1, int(code.OpAdd), 0)),
		},
		{
			// A jump to the first local can still fuse them
			"jump to the first local",
			join(code.Make(code.OpJump, 3), code.Make(code.OpGetLocal, 0), code.Make(code.OpGetLocal, 1), code.Make(code.OpAdd)),
			join(code.Make(code.OpJump, 3), code.Make(code.OpLocalLocalBinary, 0, 1, int(code.OpAdd), 0)),
		},
	}
	for _, tt := range tests {
		fuse(tt.in)
		if !bytes.Equal(tt.in, tt.want) {
			t.Errorf("%s:\ngot\n%swant\n%s", tt.name, tt.in, tt.want)
		}
	}
}
