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
			// x + (if c { 1 } else { 2 }): the jump to OpAdd keeps it
			"jump to the second instruction",
			join(code.Make(code.OpJump, 6), code.Make(code.OpConstant, 0), code.Make(code.OpAdd)),
			join(code.Make(code.OpJump, 6), code.Make(code.OpConstant, 0), code.Make(code.OpAdd)),
		},
	}
	for _, tt := range tests {
		fuse(tt.in)
		if !bytes.Equal(tt.in, tt.want) {
			t.Errorf("%s:\ngot\n%swant\n%s", tt.name, tt.in, tt.want)
		}
	}
}
