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
