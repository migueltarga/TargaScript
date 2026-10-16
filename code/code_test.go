package code

import "testing"

func TestMake(t *testing.T) {
	tests := []struct {
		op       Opcode
		operands []int
		expected []byte
	}{
		{OpConstant, []int{65534}, []byte{byte(OpConstant), 255, 254}},
		{OpAdd, []int{}, []byte{byte(OpAdd)}},
		{OpGetFree, []int{255}, []byte{byte(OpGetFree), 255}},
		{OpClosure, []int{65534, 255}, []byte{byte(OpClosure), 255, 254, 255}},
		{OpLoopEnter, []int{1, 258}, []byte{byte(OpLoopEnter), 0, 1, 1, 2}},
	}

	for _, tt := range tests {
		instruction := Make(tt.op, tt.operands...)
		if string(instruction) != string(tt.expected) {
			t.Errorf("Make(%d, %v) = %v, want %v", tt.op, tt.operands, instruction, tt.expected)
		}
	}
}

func TestInstructionsString(t *testing.T) {
	instructions := []Instructions{
		Make(OpAdd),
		Make(OpGetLocal, 1),
		Make(OpConstant, 2),
		Make(OpConstant, 65535),
		Make(OpClosure, 65535, 255),
		Make(OpCallMethod, 3, 2, 7),
	}

	expected := `0000 OpAdd
0001 OpGetLocal 1
0004 OpConstant 2
0007 OpConstant 65535
0010 OpClosure 65535 255
0014 OpCallMethod 3 2 7
`

	concatted := Instructions{}
	for _, ins := range instructions {
		concatted = append(concatted, ins...)
	}

	if concatted.String() != expected {
		t.Errorf("instructions wrongly formatted.\nwant=%q\ngot=%q", expected, concatted.String())
	}
}

func TestReadOperands(t *testing.T) {
	tests := []struct {
		op        Opcode
		operands  []int
		bytesRead int
	}{
		{OpConstant, []int{65535}, 2},
		{OpGetFree, []int{255}, 1},
		{OpClosure, []int{65535, 255}, 3},
	}

	for _, tt := range tests {
		instruction := Make(tt.op, tt.operands...)

		def, err := Lookup(byte(tt.op))
		if err != nil {
			t.Fatalf("definition not found: %q", err)
		}

		operandsRead, n := ReadOperands(def, instruction[1:])
		if n != tt.bytesRead {
			t.Fatalf("n wrong. want=%d, got=%d", tt.bytesRead, n)
		}

		for i, want := range tt.operands {
			if operandsRead[i] != want {
				t.Errorf("operand wrong. want=%d, got=%d", want, operandsRead[i])
			}
		}
	}
}

func TestEveryOpcodeIsDefined(t *testing.T) {
	for op := OpConstant; op <= OpThrow; op++ {
		if _, err := Lookup(byte(op)); err != nil {
			t.Errorf("opcode %d has no definition", op)
		}
	}
}
