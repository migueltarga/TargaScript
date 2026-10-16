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
	t, ok := readTime(values[0], values[1])
	if !ok {
		return &object.Error{Message: fmt.Sprintf("cannot read %q as a time in the layout %q", values[0], values[1])}
	}
	return &object.Integer{Value: t.Unix()}
}

// readTime reads text written in a layout of timeFormat's tokens, in the
// local time zone. It reads the tokens itself instead of using time.Parse,
// which would add 15 KB to the WebAssembly build.
func readTime(text, layout string) (time.Time, bool) {
	year, month, day, hour, minute, second := 1970, 1, 1, 0, 0, 0
	pm, hasAMPM := false, false

	// number reads n digits
	number := func(n int) (int, bool) {
		if len(text) < n {
			return 0, false
		}
		value := 0
		for _, c := range text[:n] {
			if c < '0' || c > '9' {
				return 0, false
			}
			value = value*10 + int(c-'0')
		}
		text = text[n:]
		return value, true
	}
	// name reads one of names, ignoring case, and returns its position
	name := func(names []string) (int, bool) {
		for i, n := range names {
			if len(text) >= len(n) && strings.EqualFold(text[:len(n)], n) {
				text = text[len(n):]
				return i, true
			}
		}
		return 0, false
	}

	for len(layout) > 0 {
		token := ""
		for _, t := range dateTokens {
			if strings.HasPrefix(layout, t) {
				token = t
				break
			}
		}
		ok := true
		switch token {
		case "":
			// Text outside the tokens must be there as written
			if len(text) == 0 || text[0] != layout[0] {
				return time.Time{}, false
			}
			text, layout = text[1:], layout[1:]
			continue
		case "YYYY":
			year, ok = number(4)
		case "YY":
			year, ok = number(2)
			if year < 69 {
				year += 2000
			} else {
				year += 1900
			}
		case "MMMM":
			month, ok = name(monthNames)
			month++
		case "MMM":
			month, ok = name(shortMonthNames)
			month++
		case "MM":
			month, ok = number(2)
		case "DD":
			day, ok = number(2)
		case "dddd":
			_, ok = name(dayNames)
		case "ddd":
			_, ok = name(shortDayNames)
		case "HH", "hh":
			hour, ok = number(2)
		case "mm":
			minute, ok = number(2)
		case "ss":
			second, ok = number(2)
		case "A":
			var i int
			i, ok = name([]string{"AM", "PM"})
			pm, hasAMPM = i == 1, true
		}
		if !ok {
			return time.Time{}, false
		}
		layout = layout[len(token):]
	}
	if len(text) > 0 {
		return time.Time{}, false
	}

	if hasAMPM {
		if hour < 1 || hour > 12 {
			return time.Time{}, false
		}
		hour %= 12
		if pm {
			hour += 12
		}
	}
	if month < 1 || month > 12 || day < 1 || day > 31 || hour > 23 || minute > 59 || second > 59 {
		return time.Time{}, false
	}
	t := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)
	// time.Date moves 31 February to March; that date doesn't exist
	if t.Day() != day {
		return time.Time{}, false
	}
	return t, true
}

var (
	monthNames      = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	shortMonthNames = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	dayNames        = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	shortDayNames   = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
)

// clockBuiltin returns the milliseconds since the program started, for
// timing code.
func clockBuiltin(args ...object.Object) object.Object {
	if len(args) != 0 {
		return &object.Error{Message: fmt.Sprintf("wrong number of arguments for clock. got=%d, want=0", len(args))}
	}
	return &object.Float{Value: float64(time.Since(started).Microseconds()) / 1000}
}

// dateTokens are the parts of a layout that timeFormat and parseTime
// understand, longest first so "YYYY" wins over "YY".
var dateTokens = []string{"YYYY", "YY", "MMMM", "MMM", "MM", "DD", "dddd", "ddd", "HH", "hh", "mm", "ss", "A"}

// formatTime writes t in a layout like "YYYY-MM-DD HH:mm", keeping the text
// between the tokens. It doesn't use time.Format, which would add 10 KB to
// the WebAssembly build.
func formatTime(t time.Time, layout string) string {
	var out strings.Builder
	two := func(n int) {
		out.WriteByte(byte('0' + n/10%10))
		out.WriteByte(byte('0' + n%10))
	}
	for len(layout) > 0 {
		token := ""
		for _, tok := range dateTokens {
			if strings.HasPrefix(layout, tok) {
				token = tok
				break
			}
		}
		switch token {
		case "":
			out.WriteByte(layout[0])
			layout = layout[1:]
			continue
		case "YYYY":
			two(t.Year() / 100)
			two(t.Year())
		case "YY":
			two(t.Year())
		case "MMMM":
			out.WriteString(monthNames[t.Month()-1])
		case "MMM":
			out.WriteString(shortMonthNames[t.Month()-1])
		case "MM":
			two(int(t.Month()))
		case "DD":
			two(t.Day())
		case "dddd":
			out.WriteString(dayNames[t.Weekday()])
		case "ddd":
			out.WriteString(shortDayNames[t.Weekday()])
		case "HH":
			two(t.Hour())
		case "hh":
			two((t.Hour()+11)%12 + 1)
		case "mm":
			two(t.Minute())
		case "ss":
			two(t.Second())
		case "A":
			if t.Hour() < 12 {
				out.WriteString("AM")
			} else {
				out.WriteString("PM")
			}
		}
		layout = layout[len(token):]
	}
	return out.String()
}
