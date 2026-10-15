package evaluator

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestLoopControlOutsideLoop(t *testing.T) {
	tests := []struct {
		input           string
		expectedMessage string
	}{
		{"fn stop() { break }\nrepeat i in 5 { stop() }", "break outside of a loop"},
		{"fn skip() { continue }\nrepeat i in 5 { skip() }", "continue outside of a loop"},
		{"break", "break outside of a loop"},
		{"if true { continue }", "continue outside of a loop"},
	}

	for _, tt := range tests {
		errObj, ok := testEval(tt.input).(*object.Error)
		if !ok {
			t.Errorf("%q: no error object returned", tt.input)
			continue
		}
		if errObj.Message != tt.expectedMessage {
			t.Errorf("%q: wrong error message. expected=%q, got=%q",
				tt.input, tt.expectedMessage, errObj.Message)
		}
		if errObj.Line == 0 {
			t.Errorf("%q: error has no line number", tt.input)
		}
	}

	testIntegerObject(t, testEval(`
let count = 0
repeat i in 10 {
  if i > 3 { break }
  count = count + 1
}
count
`), 3, "break inside if inside loop")
}
