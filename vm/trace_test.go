package vm

import (
	"reflect"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func call(function string, line int) object.Call {
	return object.Call{Function: function, Line: line}
}

func TestErrorTrace(t *testing.T) {
	tests := []struct {
		input string
		trace []object.Call
	}{
		{"1 / 0", []object.Call{call("", 1)}},
		{"fn f() {\n  1 / 0\n}\n\nf()", []object.Call{call("f", 2), call("", 5)}},
		{"fn inner() { null.x }\nfn outer() {\n  inner()\n}\nouter()", []object.Call{call("inner", 1), call("outer", 3), call("", 5)}},
		// Through a builtin that calls back
		{"fn f(x) { x.y.z }\n[1].map(f)", []object.Call{call("f", 1), call("", 2)}},
	}
	for _, tt := range tests {
		err, ok := testEval(tt.input).(*object.Error)
		if !ok {
			t.Errorf("%q: no error", tt.input)
			continue
		}
		if !reflect.DeepEqual(err.Trace, tt.trace) {
			t.Errorf("%q: trace %v, want %v", tt.input, err.Trace, tt.trace)
		}
	}
}

func TestCaughtErrorsAreNotTraced(t *testing.T) {
	result := testEval("fn f() { 1 / 0 }\ntry { f() } catch e { e.message }")
	if str, ok := result.(*object.String); !ok || str.Value != "division by zero" {
		t.Errorf("got %s", result.Inspect())
	}
}
