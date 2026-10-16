package evaluator

import (
	"fmt"
	"strings"
	"time"

	"github.com/migueltarga/TargaScript/object"
)

var started = time.Now()

func init() {
	builtins["clock"] = &object.Builtin{Name: "clock", Fn: clockBuiltin}
	builtins["parseTime"] = &object.Builtin{Name: "parseTime", Fn: parseTimeBuiltin}
}

// parseTimeBuiltin reads a time written with a layout like timeFormat's and
// returns its Unix time in seconds, in the local time zone unless the
// layout has one.
func parseTimeBuiltin(args ...object.Object) object.Object {
	values, err := stringArgs("parseTime", args, 2)
	if err != nil {
		return err
	}
	t, parseErr := time.ParseInLocation(dateLayout(values[1]), values[0], time.Local)
	if parseErr != nil {
		return &object.Error{Message: fmt.Sprintf("cannot read %q as a time in the layout %q", values[0], values[1])}
	}
	return &object.Integer{Value: t.Unix()}
}

// clockBuiltin returns the milliseconds since the program started, for
// timing code.
func clockBuiltin(args ...object.Object) object.Object {
	if len(args) != 0 {
		return &object.Error{Message: fmt.Sprintf("wrong number of arguments for clock. got=%d, want=0", len(args))}
	}
	return &object.Float{Value: float64(time.Since(started).Microseconds()) / 1000}
}

// dateTokens maps the tokens timeFormat understands to Go layout parts,
// longest first so "YYYY" wins over "YY".
var dateTokens = []struct{ token, layout string }{
	{"YYYY", "2006"},
	{"YY", "06"},
	{"MMMM", "January"},
	{"MMM", "Jan"},
	{"MM", "01"},
	{"DD", "02"},
	{"dddd", "Monday"},
	{"ddd", "Mon"},
	{"HH", "15"},
	{"hh", "03"},
	{"mm", "04"},
	{"ss", "05"},
	{"A", "PM"},
}

// dateLayout turns a layout like "YYYY-MM-DD HH:mm" into the Go layout.
// Text without tokens is kept, so Go layouts like "2006-01-02" still work.
func dateLayout(format string) string {
	var out strings.Builder
	for i := 0; i < len(format); {
		matched := false
		for _, t := range dateTokens {
			if strings.HasPrefix(format[i:], t.token) {
				out.WriteString(t.layout)
				i += len(t.token)
				matched = true
				break
			}
		}
		if !matched {
			out.WriteByte(format[i])
			i++
		}
	}
	return out.String()
}
