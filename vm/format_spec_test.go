package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestInterpolationFormats(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`let price = 12.5` + "\n\"Total: ${price:.2f}\"", "Total: 12.50"},
		{`"[${42:5d}][${"ab":-4s}][${7:03d}]"`, "[   42][ab  ][007]"},
		{`"${255:x} ${3.14159:.3f} ${1:+d}"`, "ff 3.142 +1"},
		{`let h = {"a": 1}` + "\n\"${h[\"a\"]:2d}|${ {\"b\": 2}.b }\"", " 1|2"},
		{`let format = "shadowed"` + "\n\"${1.5:.1f} ${format}\"", "1.5 shadowed"},
		{`"${[1, 2].sum():4d}"`, "   3"},
		{`"${"text":d}"`, "ERROR: printf verb %d does not accept a string"},
		{`"${1:zz}"`, `ERROR: parser errors: line 1:1: in ${1:zz}: "zz" is not a format like .2f, 5d or -10s`},
		{`"${1:}"`, `ERROR: parser errors: line 1:1: in ${1:}: "" is not a format like .2f, 5d or -10s`},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		got := result.Inspect()
		if s, ok := result.(*object.String); ok {
			got = s.Value
		}
		if err, ok := result.(*object.Error); ok {
			got = "ERROR: " + err.Message
		}
		if got != tt.expected {
			t.Errorf("%s\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
