// Package code defines the bytecode instructions run by the virtual
// machine: one byte for the opcode followed by its operands, big endian.
package code

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Instructions []byte

type Opcode byte

const (
	// Constants and literals
	OpConstant Opcode = iota // push constants[index]
	OpTrue
	OpFalse
	OpNull
	OpArray    // pop n elements, push an array
	OpHash     // pop n key/value pairs, push an object
	OpTemplate // pop n parts, push them joined as a string

	// Operators
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpXor
	OpEqual
	OpNotEqual
	OpGreaterThan
	OpGreaterEqual
	OpLessThan
	OpLessEqual
	OpRange // pop end and start, push the array start..end
	OpMinus
	OpBang
	OpIncrement // add 1 to the number on top, or subtract 1 when the operand is 1

	// Stack
	OpPop
	OpDup  // push a copy of the top value
	OpDup2 // push copies of the two top values

	// Jumps
	OpJump              // jump to the operand
	OpJumpNotTruthy     // pop; jump if falsy
	OpJumpNotTruthyKeep // jump if the top is falsy, keeping it; else pop it
	OpJumpTruthyKeep    // jump if the top is truthy, keeping it; else pop it
	OpLoopEnter         // remember the stack height and the break and continue targets
	OpLoopExit          // forget the innermost loop
	OpBreak             // restore the loop's stack height and jump to its end, or to its next iteration when the operand is 1
	OpIterStart         // pop a collection, push an iterator over it
	OpIterNext          // push the iterator's next value, or jump when done
	OpRangeIter         // pop end and start, push an iterator over start..end

	// Variables
	OpGetGlobal
	OpSetGlobal // store the top in a global, keeping it
	OpGetLocal
	OpSetLocal // store the top in a local, keeping it
	OpGetFree
	OpSetFree      // store the top in a captured variable, keeping it
	OpNewCell      // store the top in a new cell in a local, keeping it
	OpGetLocalCell // push the cell of a captured local, to capture it again
	OpGetFreeCell  // push the cell of a free variable, to capture it again
	OpGetBuiltin

	// Objects
	OpIndex       // pop index and container, push container[index]
	OpSetIndex    // pop value, index, container; store; push value
	OpGetProperty // pop object, push object.name
	OpSetProperty // pop value, object; store object.name; push value

	// Calls
	OpCall       // call the function below the arguments
	OpCallMethod // call the method named by a constant on the object below the arguments
	OpReturnValue
	OpClosure // push a closure of constants[index] capturing n cells
	OpPrint   // pop n values and print them
	OpLoad    // run the file at constants[path] once and push an object of its top-level names
	OpError   // stop with the error message in constants[index]
	OpAwait   // pop a value; if it's a promise, pause until it settles and push its value
)

// Definition describes an opcode: its name and the width in bytes of each
// operand.
type Definition struct {
	Name          string
	OperandWidths []int
}

var definitions = map[Opcode]*Definition{
	OpConstant: {"OpConstant", []int{2}},
	OpTrue:     {"OpTrue", []int{}},
	OpFalse:    {"OpFalse", []int{}},
	OpNull:     {"OpNull", []int{}},
	OpArray:    {"OpArray", []int{2}},
	OpHash:     {"OpHash", []int{2}},
	OpTemplate: {"OpTemplate", []int{2}},

	OpAdd:          {"OpAdd", []int{}},
	OpSub:          {"OpSub", []int{}},
	OpMul:          {"OpMul", []int{}},
	OpDiv:          {"OpDiv", []int{}},
	OpMod:          {"OpMod", []int{}},
	OpXor:          {"OpXor", []int{}},
	OpEqual:        {"OpEqual", []int{}},
	OpNotEqual:     {"OpNotEqual", []int{}},
	OpGreaterThan:  {"OpGreaterThan", []int{}},
	OpGreaterEqual: {"OpGreaterEqual", []int{}},
	OpLessThan:     {"OpLessThan", []int{}},
	OpLessEqual:    {"OpLessEqual", []int{}},
	OpRange:        {"OpRange", []int{}},
	OpMinus:        {"OpMinus", []int{}},
	OpBang:         {"OpBang", []int{}},
	OpIncrement:    {"OpIncrement", []int{1}},

	OpPop:  {"OpPop", []int{}},
	OpDup:  {"OpDup", []int{}},
	OpDup2: {"OpDup2", []int{}},

	OpJump:              {"OpJump", []int{2}},
	OpJumpNotTruthy:     {"OpJumpNotTruthy", []int{2}},
	OpJumpNotTruthyKeep: {"OpJumpNotTruthyKeep", []int{2}},
	OpJumpTruthyKeep:    {"OpJumpTruthyKeep", []int{2}},
	OpLoopEnter:         {"OpLoopEnter", []int{2, 2}},
	OpLoopExit:          {"OpLoopExit", []int{}},
	OpBreak:             {"OpBreak", []int{1}},
	OpIterStart:         {"OpIterStart", []int{}},
	OpIterNext:          {"OpIterNext", []int{2}},
	OpRangeIter:         {"OpRangeIter", []int{}},

	OpGetGlobal:    {"OpGetGlobal", []int{2}},
	OpSetGlobal:    {"OpSetGlobal", []int{2}},
	OpGetLocal:     {"OpGetLocal", []int{2}},
	OpSetLocal:     {"OpSetLocal", []int{2}},
	OpGetFree:      {"OpGetFree", []int{1}},
	OpSetFree:      {"OpSetFree", []int{1}},
	OpNewCell:      {"OpNewCell", []int{2}},
	OpGetLocalCell: {"OpGetLocalCell", []int{2}},
	OpGetFreeCell:  {"OpGetFreeCell", []int{1}},
	OpGetBuiltin:   {"OpGetBuiltin", []int{2}},

	OpIndex:       {"OpIndex", []int{}},
	OpSetIndex:    {"OpSetIndex", []int{}},
	OpGetProperty: {"OpGetProperty", []int{2}},
	OpSetProperty: {"OpSetProperty", []int{2}},

	OpCall:        {"OpCall", []int{1}},
	OpCallMethod:  {"OpCallMethod", []int{2, 1}},
	OpReturnValue: {"OpReturnValue", []int{}},
	OpClosure:     {"OpClosure", []int{2, 1}},
	OpPrint:       {"OpPrint", []int{1}},
	OpLoad:        {"OpLoad", []int{2, 2}},
	OpError:       {"OpError", []int{2}},
	OpAwait:       {"OpAwait", []int{}},
}

// Lookup returns the definition of an opcode.
func Lookup(op byte) (*Definition, error) {
	def, ok := definitions[Opcode(op)]
	if !ok {
		return nil, fmt.Errorf("opcode %d undefined", op)
	}
	return def, nil
}

// Make encodes an instruction.
func Make(op Opcode, operands ...int) []byte {
	def, ok := definitions[op]
	if !ok {
		return []byte{}
	}

	length := 1
	for _, w := range def.OperandWidths {
		length += w
	}

	instruction := make([]byte, length)
	instruction[0] = byte(op)

	offset := 1
	for i, o := range operands {
		width := def.OperandWidths[i]
		switch width {
		case 2:
			binary.BigEndian.PutUint16(instruction[offset:], uint16(o))
		case 1:
			instruction[offset] = byte(o)
		}
		offset += width
	}
	return instruction
}

// ReadOperands decodes the operands of an instruction, returning them and
// the number of bytes read.
func ReadOperands(def *Definition, ins Instructions) ([]int, int) {
	operands := make([]int, len(def.OperandWidths))
	offset := 0
	for i, width := range def.OperandWidths {
		switch width {
		case 2:
			operands[i] = int(ReadUint16(ins[offset:]))
		case 1:
			operands[i] = int(ReadUint8(ins[offset:]))
		}
		offset += width
	}
	return operands, offset
}

func ReadUint16(ins Instructions) uint16 {
	return binary.BigEndian.Uint16(ins)
}

func ReadUint8(ins Instructions) uint8 {
	return uint8(ins[0])
}

// String disassembles the instructions, one per line with its offset.
func (ins Instructions) String() string {
	var out bytes.Buffer

	for i := 0; i < len(ins); {
		def, err := Lookup(ins[i])
		if err != nil {
			fmt.Fprintf(&out, "ERROR: %s\n", err)
			i++
			continue
		}

		operands, read := ReadOperands(def, ins[i+1:])
		fmt.Fprintf(&out, "%04d %s\n", i, formatInstruction(def, operands))
		i += 1 + read
	}

	return out.String()
}

func formatInstruction(def *Definition, operands []int) string {
	if len(operands) != len(def.OperandWidths) {
		return fmt.Sprintf("ERROR: operand len %d does not match defined %d\n", len(operands), len(def.OperandWidths))
	}

	switch len(operands) {
	case 0:
		return def.Name
	case 1:
		return fmt.Sprintf("%s %d", def.Name, operands[0])
	case 2:
		return fmt.Sprintf("%s %d %d", def.Name, operands[0], operands[1])
	}
	return fmt.Sprintf("ERROR: unhandled operand count for %s\n", def.Name)
}
