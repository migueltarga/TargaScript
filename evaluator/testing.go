package evaluator

import (
	"fmt"

	"github.com/migueltarga/TargaScript/object"
)

// Counts of the tests run with test since the last ResetTests
var testsPassed, testsFailed int

func init() {
	builtins["test"] = &object.Builtin{Name: "test", Fn: testBuiltin}
}

// testBuiltin runs a function as a named test and prints whether it
// passed. A failing test doesn't stop the program, so the rest still run.
func testBuiltin(args ...object.Object) object.Object {
	if len(args) != 2 {
		return &object.Error{Message: fmt.Sprintf("wrong number of arguments for test. got=%d, want=2", len(args))}
	}
	name, ok := args[0].(*object.String)
	if !ok {
		return &object.Error{Message: fmt.Sprintf("first argument to `test` must be a string, got %s", typeName(args[0]))}
	}
	if !isCallable(args[1]) {
		return &object.Error{Message: fmt.Sprintf("second argument to `test` must be a function, got %s", typeName(args[1]))}
	}

	result := applyFunction(args[1], nil)
	if err, ok := result.(*object.Error); ok {
		testsFailed++
		where := ""
		if err.Line > 0 {
			where = fmt.Sprintf(" (line %d)", err.Line)
		}
		fmt.Fprintf(output, "✗ %s\n    %s%s\n", name.Value, err.Message, where)
		return NULL
	}
	testsPassed++
	fmt.Fprintf(output, "✓ %s\n", name.Value)
	return NULL
}

// TestResults returns how many tests passed and failed since the last
// ResetTests.
func TestResults() (passed, failed int) {
	return testsPassed, testsFailed
}

// ResetTests sets the counts of TestResults back to zero.
func ResetTests() {
	testsPassed, testsFailed = 0, 0
}
