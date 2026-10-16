package strfmt

import (
	"errors"
	"fmt"
	"math"
	"testing"
)

type named struct{}

func (named) String() string { return "named" }

// Sprintf gives what fmt.Sprintf gives, for every verb, flag and precision
// it knows. printf in TargaScript passes these on, and applies a width
// itself.
func TestLikeFmt(t *testing.T) {
	values := []any{
		0, 7, -7, 42, 255, -255, int64(math.MaxInt64), int64(math.MinInt64), int32('é'),
		0.0, 1.5, -1.5, 1234.5678, 1e21, 1e-7, math.Inf(1), math.Inf(-1), math.NaN(), float32(2.5),
		"", "a", "héllo", "say \"hi\"\n", "tab\there", "`back`",
		true, false, errors.New("an error"), named{},
	}
	verbs := []string{"v", "s", "q", "d", "b", "o", "x", "X", "c", "t", "e", "E", "f", "F", "g", "G"}
	flags := []string{"", "+", "-", "#", " ", "0", "+0", "-0", "#0", "+#"}
	widths := []string{""}
	precisions := []string{"", ".", ".0", ".2", ".5", ".*"}
	supported := func(verb string, v any) bool {
		switch v.(type) {
		case string:
			return verb == "v" || verb == "s" || verb == "q" || verb == "x" || verb == "X"
		case error, named:
			// %#v of these is Go syntax, which nothing uses
			return verb == "s" || verb == "q" || verb == "x" || verb == "X"
		case bool:
			return verb == "v" || verb == "t"
		case float64, float32:
			return verb == "v" || contains("eEfFgG", verb)
		default:
			// Integers
			return verb != "t" && verb != "s" && !contains("eEfFgG", verb)
		}
	}
	cases := 0
	defer func() { t.Logf("%d formats checked", cases) }()
	for _, v := range values {
		for _, verb := range verbs {
			if !supported(verb, v) {
				continue
			}
			for _, flag := range flags {
				for _, width := range widths {
					for _, precision := range precisions {
						format := "%" + flag + width + precision + verb
						args := []any{}
						if width == "*" {
							args = append(args, 6)
						}
						if precision == ".*" {
							args = append(args, 3)
						}
						args = append(args, v)
						cases++
						want := fmt.Sprintf(format, args...)
						if got := Sprintf(format, args...); got != want {
							t.Errorf("Sprintf(%q, %#v) = %q, want %q", format, v, got, want)
						}
					}
				}
			}
		}
	}
}

func contains(set, s string) bool {
	for _, c := range set {
		if string(c) == s {
			return true
		}
	}
	return false
}

// Widths, as the messages of the code use them
func TestWidths(t *testing.T) {
	for _, format := range []string{"%4d", "%04d", "%05d", "%-5d", "%04x", "%-12s", "%12s", "%8.2f", "%-8.2f", "%*d", "%-*s", "%3s"} {
		for _, v := range []any{0, 7, -42, 123456789, "ab", "héllo", 3.14159} {
			_, isString := v.(string)
			_, isFloat := v.(float64)
			verb := format[len(format)-1]
			if (verb == 's') != isString || (verb == 'f') != isFloat {
				continue
			}
			args := []any{v}
			if format[1] == '*' || format[2] == '*' {
				args = []any{6, v}
			}
			want := fmt.Sprintf(format, args...)
			if got := Sprintf(format, args...); got != want {
				t.Errorf("Sprintf(%q, %#v) = %q, want %q", format, v, got, want)
			}
		}
	}
}

// The forms the code uses, mixed in one format
func TestMessages(t *testing.T) {
	tests := []struct {
		format string
		args   []any
	}{
		{"wrong number of arguments: got %d, want %d", []any{1, 2}},
		{"%s can't be an object key: %q", []any{"array", "x"}},
		{"line %04d col %05d %04x %-12s|", []any{7, -3, 255, "name"}},
		{"100%% done, %c%c", []any{'o', 'k'}},
		{"%v %v %v %v", []any{1, "a", true, 2.5}},
		{"%s", []any{errors.New("boom")}},
	}
	for _, tt := range tests {
		if got, want := Sprintf(tt.format, tt.args...), fmt.Sprintf(tt.format, tt.args...); got != want {
			t.Errorf("Sprintf(%q) = %q, want %q", tt.format, got, want)
		}
	}
	if err := Errorf("x %d", 1); err.Error() != "x 1" {
		t.Errorf("Errorf gave %q", err)
	}
}
