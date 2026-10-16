package evaluator

import (
	"strings"

	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/strfmt"
)

// Counts of the tests run with test since the last ResetTests
var testsPassed, testsFailed, testsSkipped int

// testGroups are the names of the describe calls running, outermost first,
// and groupsShown how many of them have been printed. A group is printed
// before its first test that runs, so -run leaves out empty groups.
var (
	testGroups  []string
	groupsShown int
)

// testFilter, when set, is text a test's name must contain for it to run
var testFilter string

// SetTestFilter makes test run only the tests whose name contains text,
// ignoring case. An empty text runs them all.
func SetTestFilter(text string) {
	testFilter = strings.ToLower(text)
}

func init() {
	builtins["test"] = &object.Builtin{Name: "test", Fn: testBuiltin}
	builtins["describe"] = &object.Builtin{Name: "describe", Fn: describeBuiltin}
}

// describeBuiltin runs a function that defines tests, printing its name
// above them. Groups nest, and their names are part of the names of their
// tests for -run.
func describeBuiltin(args ...object.Object) object.Object {
	if len(args) != 2 {
		return wrongArgs(0, 0, "describe", len(args), "2")
	}
	name, ok := args[0].(*object.String)
	if !ok {
		return &object.Error{Message: strfmt.Sprintf("first argument to `describe` must be a string, got %s", typeName(args[0]))}
	}
	if !isCallable(args[1]) {
		return &object.Error{Message: strfmt.Sprintf("second argument to `describe` must be a function, got %s", typeName(args[1]))}
	}

	testGroups = append(testGroups, name.Value)
	result := applyFunctionAsync(args[1], nil)
	if promise, ok := result.(*object.Promise); ok && promise.Paused {
		done := &object.Promise{Paused: true}
		promise.Then(func(object.Object) {
			leaveTestGroup()
			done.Resolve(NULL)
		}, func(reason object.Object) {
			leaveTestGroup()
			done.Reject(reason)
		})
		return done
	}
	leaveTestGroup()
	if err, ok := result.(*object.Error); ok {
		return err
	}
	return NULL
}

func leaveTestGroup() {
	testGroups = testGroups[:len(testGroups)-1]
	groupsShown = min(groupsShown, len(testGroups))
}

// testIndent is the space before the lines of a test in the groups running.
func testIndent() string {
	return strings.Repeat("  ", len(testGroups))
}

// testBuiltin runs a function as a named test and prints whether it
// passed. A failing test doesn't stop the program, so the rest still run.
func testBuiltin(args ...object.Object) object.Object {
	if len(args) != 2 {
		return wrongArgs(0, 0, "test", len(args), "2")
	}
	name, ok := args[0].(*object.String)
	if !ok {
		return &object.Error{Message: strfmt.Sprintf("first argument to `test` must be a string, got %s", typeName(args[0]))}
	}
	if !isCallable(args[1]) {
		return &object.Error{Message: strfmt.Sprintf("second argument to `test` must be a function, got %s", typeName(args[1]))}
	}

	fullName := strings.Join(append(append([]string{}, testGroups...), name.Value), " ")
	if testFilter != "" && !strings.Contains(strings.ToLower(fullName), testFilter) {
		testsSkipped++
		return NULL
	}

	// A test that awaits gives a promise; the program waits for the one
	// returned here, which settles once the test is reported, whether it
	// passed or failed
	result := applyFunctionAsync(args[1], nil)
	if promise, ok := result.(*object.Promise); ok && promise.Paused {
		reported := &object.Promise{Paused: true}
		promise.Then(func(object.Object) {
			reportTest(name.Value, nil)
			reported.Resolve(NULL)
		}, func(reason object.Object) {
			err, ok := reason.(*object.Error)
			if !ok {
				err = &object.Error{Message: objectToString(reason)}
			}
			reportTest(name.Value, err)
			reported.Resolve(NULL)
		})
		return reported
	}
	err, _ := result.(*object.Error)
	reportTest(name.Value, err)
	return NULL
}

// reportTest prints whether a test passed, and counts it.
func reportTest(name string, err *object.Error) {
	for ; groupsShown < len(testGroups); groupsShown++ {
		strfmt.Fprintf(output, "%s%s\n", strings.Repeat("  ", groupsShown), testGroups[groupsShown])
	}
	if err == nil {
		testsPassed++
		strfmt.Fprintf(output, "%s✓ %s\n", testIndent(), name)
		return
	}
	testsFailed++
	where := ""
	if err.Line > 0 {
		where = strfmt.Sprintf(" (line %d)", err.Line)
	}
	indent := testIndent()
	strfmt.Fprintf(output, "%s✗ %s\n%s    %s%s\n", indent, name, indent, err.Message, where)
}

// TestResults returns how many tests passed, failed and were left out by
// the filter since the last ResetTests.
func TestResults() (passed, failed, skipped int) {
	return testsPassed, testsFailed, testsSkipped
}

// ResetTests sets the counts of TestResults back to zero.
func ResetTests() {
	testsPassed, testsFailed, testsSkipped = 0, 0, 0
	testGroups, groupsShown = nil, 0
}
