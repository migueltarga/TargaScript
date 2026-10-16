package vm

import (
	"bytes"
	"github.com/migueltarga/TargaScript/evaluator"
	"os"
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestPrintfFormatting(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`printf("%d and %s", 42, "text")`, "42 and text\n"},
		{`printf("100%%")`, "100%\n"},
		{`printf("%.2f %f", 3.14159, 2)`, "3.14 2.000000\n"},
		{`printf("|%5d|%-4s|%05.1f|", 42, "ab", 3.14)`, "|   42|ab  |003.1|\n"},
		{`printf("|%*d|%.*f|", 4, 7, 1, 2.25)`, "|   7|2.2|\n"},
		{`printf("%s %v", 5, [1, "a"])`, "5 [1, \"a\"]\n"},
		{`printf("%v %v %s", null, true, null)`, "null true null\n"},
		{`printf("%t %x %X %q", false, 255, "hi", "q")`, "false ff 6869 \"q\"\n"},
		{`printf("café %d", 1)`, "café 1\n"},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		evaluator.SetOutput(&out)
		result := testEval(tt.input)
		evaluator.SetOutput(os.Stdout)

		if isError(result) {
			t.Errorf("%s: unexpected error %s", tt.input, result.Inspect())
			continue
		}
		if out.String() != tt.expected {
			t.Errorf("%s: got=%q, want=%q", tt.input, out.String(), tt.expected)
		}
	}
}

func TestPrintfErrors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`printf("%d %d", 1)`, "not enough arguments for printf"},
		{`printf("%s")`, "not enough arguments for printf"},
		{`printf("%d", 1, 2)`, "too many arguments for printf: format uses 1 of the 2 given"},
		{`printf("plain", 1)`, "too many arguments for printf"},
		{`printf("%d", "x")`, "printf verb %d does not accept STRING"},
		{`printf("%d", 1.5)`, "printf verb %d does not accept FLOAT"},
		{`printf("%t", 1)`, "printf verb %t does not accept INTEGER"},
		{`printf("%y", 1)`, "unknown printf verb %y"},
		{`printf("%*d", "w", 1)`, "printf width given by * must be an INTEGER"},
		{`printf("50%")`, "printf format ends with an incomplete verb"},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		evaluator.SetOutput(&out)
		result := testEval(tt.input)
		evaluator.SetOutput(os.Stdout)

		errObj, ok := result.(*object.Error)
		if !ok {
			t.Errorf("%s: expected an error, got %s (output %q)", tt.input, result.Inspect(), out.String())
			continue
		}
		if !strings.Contains(errObj.Message, tt.expected) {
			t.Errorf("%s: got=%q, want it to contain %q", tt.input, errObj.Message, tt.expected)
		}
		if out.Len() != 0 {
			t.Errorf("%s: printed %q before failing", tt.input, out.String())
		}
	}
}
