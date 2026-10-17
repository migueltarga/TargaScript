// Package code defines the instructions of programs: the opcodes of the
// stack code the compiler makes first, and the register code it translates
// that to, which the virtual machine runs.
package code

// Opcode is an instruction of stack code.
type Opcode byte

const (
	// Constants and literals
	OpConstant Opcode = iota // push constants[index]
	OpTrue
	OpFalse
	OpNull
	OpArray     // pop n elements, push an array
	OpHash      // pop n key/value pairs, push an object; the second operand is the cache of its shape
	OpTemplate  // pop n parts, push them joined as a string
	OpAppend    // pop a value and add it to the array below, or its elements when the operand is 1
	OpHashSet   // pop a value and a key and set them in the object below
	OpHashMerge // pop an object and copy its properties into the object below
	OpWithout   // replace the object on top with a copy without the keys in constants[index]

	// Operators
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpPow
	OpXor
	OpBitAnd
	OpBitOr
	OpShl
	OpShr
	OpEqual
	OpNotEqual
	OpGreaterThan
	OpGreaterEqual
	OpLessThan
	OpLessEqual
	OpRange   // pop end and start, push the array start..end
	OpInRange // pop end, start and a value, push whether start <= value <= end
	OpIn      // pop a collection and a value, push whether the value is in it
	OpMinus
	OpBang
	OpIncrement // add 1 to the number on top, or subtract 1 when the operand is 1

	// Stack
	OpPop
	OpDup   // push a copy of the top value
	OpDup2  // push copies of the two top values
	OpSlide // keep the top value and remove the n values below it

	// Jumps
	OpJump              // jump to the operand
	OpJumpNotTruthy     // pop; jump if falsy
	OpJumpNotTruthyKeep // jump if the top is falsy, keeping it; else pop it
	OpJumpTruthyKeep    // jump if the top is truthy, keeping it; else pop it
	OpJumpNotNullKeep   // jump if the top isn't null, keeping it; else pop it
	OpJumpNullKeep      // jump if the top is null, keeping it
	OpLoopEnter         // remember the stack height and the break and continue targets
	OpLoopExit          // forget the innermost loop
	OpBreak             // restore the loop's stack height and jump to its end, or to its next iteration when the operand is 1
	OpIterStart         // pop a collection, push an iterator over it
	OpIterNext          // push the iterator's next value, or its key and value when the second operand is 1, or jump when done
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
	OpGetProperty // pop object, push object.name; the second operand is the property cache
	OpSetProperty // pop value, object; store object.name; push value; the second operand is the property cache

	// Calls
	OpCall            // call the function below the arguments
	OpCallMethod      // call the method named by a constant on the object below the arguments; the last operand is the property cache
	OpCallArray       // call the function below an array, with its elements as arguments
	OpCallMethodArray // call a method with the elements of the array on top as arguments
	OpReturnValue
	OpClosure // push a closure of constants[index] capturing n cells
	OpLoad    // run the file at constants[path] once and push an object of its top-level names
	OpError   // stop with the error message in constants[index]
	OpAwait   // pop a value; if it's a promise, pause until it settles and push its value
	OpTry     // start a try block whose catch block starts at the operand
	OpTryEnd  // end the innermost try block
	OpThrow   // pop a value and raise an error with it

	// Structs
	OpStruct    // pop a constructor and n methods, push a struct of the spec constants[index]
	OpStructNew // pop the values of n fields and a struct, push a value of the struct
)
