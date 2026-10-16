package evaluator

import (
	"strings"

	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/strfmt"
)

// TimeHost tells the time. Go's time package is large in the WebAssembly
// build, which asks JavaScript's Date instead and does the calendar here.
type TimeHost interface {
	// Now returns the Unix time in seconds
	Now() float64
	// Offset returns how many seconds the local time zone is ahead of UTC
	// at a Unix time
	Offset(unix int64) int64
}

var (
	timeHost TimeHost
	started  float64 // when the time host was set, for clock
)

// SetTimeHost sets what now, clock, timeFormat and parseTime use.
func SetTimeHost(host TimeHost) {
	timeHost = host
	started = host.Now()
}

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
	unix, ok := readTime(values[0], values[1])
	if !ok {
		return &object.Error{Message: strfmt.Sprintf("cannot read %q as a time in the layout %q", values[0], values[1])}
	}
	return &object.Integer{Value: unix}
}

// readTime reads text written in a layout of timeFormat's tokens, in the
// local time zone, and returns its Unix time.
func readTime(text, layout string) (int64, bool) {
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
				return 0, false
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
			return 0, false
		}
		layout = layout[len(token):]
	}
	if len(text) > 0 {
		return 0, false
	}

	if hasAMPM {
		if hour < 1 || hour > 12 {
			return 0, false
		}
		hour %= 12
		if pm {
			hour += 12
		}
	}
	if month < 1 || month > 12 || day < 1 || day > 31 || hour > 23 || minute > 59 || second > 59 {
		return 0, false
	}
	days := daysFromCivil(int64(year), int64(month), int64(day))
	// 31 February would be read as a day in March; that date doesn't exist
	if _, _, d, _ := civilFromDays(days); d != day {
		return 0, false
	}
	local := days*86400 + int64(hour*3600+minute*60+second)
	// The offset at the local time is a guess, right but near a change of
	// daylight saving time; the offset at the guess is right
	unix := local - timeHost.Offset(local)
	return local - timeHost.Offset(unix), true
}

// civilFromDays returns the date and the weekday (0 is Sunday) of a number
// of days since 1 January 1970, with the algorithms in
// https://howardhinnant.github.io/date_algorithms.html.
func civilFromDays(days int64) (year int64, month, day, weekday int) {
	weekday = int(floorMod(days+4, 7))
	z := days + 719468
	era := floorDiv(z, 146097)
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	day = int(doy - (153*mp+2)/5 + 1)
	month = int(mp + 3)
	if month > 12 {
		month -= 12
	}
	year = yoe + era*400
	if month <= 2 {
		year++
	}
	return year, month, day, weekday
}

// daysFromCivil is the opposite of civilFromDays.
func daysFromCivil(year, month, day int64) int64 {
	if month <= 2 {
		year--
	}
	era := floorDiv(year, 400)
	yoe := year - era*400
	doy := (153*((month+9)%12)+2)/5 + day - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 719468
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

func floorMod(a, b int64) int64 {
	return a - floorDiv(a, b)*b
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
		return &object.Error{Message: strfmt.Sprintf("wrong number of arguments for clock. got=%d, want=0", len(args))}
	}
	return &object.Float{Value: float64(int64((timeHost.Now()-started)*1e6)) / 1000}
}

// dateTokens are the parts of a layout that timeFormat and parseTime
// understand, longest first so "YYYY" wins over "YY".
var dateTokens = []string{"YYYY", "YY", "MMMM", "MMM", "MM", "DD", "dddd", "ddd", "HH", "hh", "mm", "ss", "A"}

// formatTime writes a Unix time in the local time zone, in a layout like
// "YYYY-MM-DD HH:mm", keeping the text between the tokens.
func formatTime(unix int64, layout string) string {
	local := unix + timeHost.Offset(unix)
	days := floorDiv(local, 86400)
	seconds := int(local - days*86400)
	year, month, day, weekday := civilFromDays(days)
	hour, minute, second := seconds/3600, seconds/60%60, seconds%60
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
			two(int(year / 100))
			two(int(year))
		case "YY":
			two(int(year))
		case "MMMM":
			out.WriteString(monthNames[month-1])
		case "MMM":
			out.WriteString(shortMonthNames[month-1])
		case "MM":
			two(month)
		case "DD":
			two(day)
		case "dddd":
			out.WriteString(dayNames[weekday])
		case "ddd":
			out.WriteString(shortDayNames[weekday])
		case "HH":
			two(hour)
		case "hh":
			two((hour+11)%12 + 1)
		case "mm":
			two(minute)
		case "ss":
			two(second)
		case "A":
			if hour < 12 {
				out.WriteString("AM")
			} else {
				out.WriteString("PM")
			}
		}
		layout = layout[len(token):]
	}
	return out.String()
}
