package evaluator

import (
	"testing"
	"time"

	"github.com/migueltarga/TargaScript/object"
)

// utc is a time host in UTC, so tests don't depend on the machine's zone
func utc(t *testing.T) {
	previous := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = previous })
}

func TestFormatTime(t *testing.T) {
	utc(t)
	stamp := time.Date(2026, time.March, 5, 14, 7, 9, 0, time.UTC).Unix()
	tests := []struct{ format, want string }{
		{"YYYY-MM-DD", "2026-03-05"},
		{"DD/MM/YY HH:mm:ss", "05/03/26 14:07:09"},
		{"hh:mm A", "02:07 PM"},
		{"dddd, MMMM DD", "Thursday, March 05"},
		{"ddd MMM", "Thu Mar"},
		{"Report 2 of DD", "Report 2 of 05"},
		{"hh A", "02 PM"},
	}
	for _, tt := range tests {
		if got := formatTime(stamp, tt.format); got != tt.want {
			t.Errorf("%q: got %q, want %q", tt.format, got, tt.want)
		}
	}
}

func TestClock(t *testing.T) {
	first, ok := clockBuiltin().(*object.Float)
	if !ok || first.Value < 0 {
		t.Fatalf("clock() = %v", clockBuiltin())
	}
	if _, ok := clockBuiltin(&object.Integer{Value: 1}).(*object.Error); !ok {
		t.Errorf("clock(1) should fail")
	}
}

func TestParseTime(t *testing.T) {
	utc(t)

	tests := []struct {
		text, layout string
		want         int64
	}{
		{"1970-01-02", "YYYY-MM-DD", 86400},
		{"02/01/1970 01:00", "DD/MM/YYYY HH:mm", 86400 + 3600},
		{"Jan 2 1970", "MMM D YYYY", -1},
		{"01/01/70 12:00:10 AM", "DD/MM/YY hh:mm:ss A", 10},
		{"Jan 01 1970 01:00 pm", "MMM DD YYYY hh:mm A", 13 * 3600},
		{"Friday, January 02 1970", "dddd, MMMM DD YYYY", 86400},
		{"1970-02-31", "YYYY-MM-DD", -1},
		{"1970-01-01 extra", "YYYY-MM-DD", -1},
		{"1970-1-01", "YYYY-MM-DD", -1},
		{"13:00 PM", "hh:mm A", -1},
	}
	for _, tt := range tests {
		result := parseTimeBuiltin(&object.String{Value: tt.text}, &object.String{Value: tt.layout})
		if tt.want == -1 {
			if _, ok := result.(*object.Error); !ok {
				t.Errorf("%q in %q: want an error, got %s", tt.text, tt.layout, result.Inspect())
			}
			continue
		}
		if n, ok := result.(*object.Integer); !ok || n.Value != tt.want {
			t.Errorf("%q in %q: got %s, want %d", tt.text, tt.layout, result.Inspect(), tt.want)
		}
	}

	// timeFormat and parseTime are opposites
	stamp := int64(1_760_000_000)
	formatted := formatTime(stamp, "YYYY-MM-DD HH:mm:ss")
	if n := parseTimeBuiltin(&object.String{Value: formatted}, &object.String{Value: "YYYY-MM-DD HH:mm:ss"}).(*object.Integer); n.Value != stamp {
		t.Errorf("round trip: got %d, want %d", n.Value, stamp)
	}
}

// The calendar arithmetic agrees with Go's time package, across leap years
// and before 1970
func TestCivilDays(t *testing.T) {
	for days := int64(-800_000); days <= 800_000; days += 37 {
		want := time.Unix(days*86400, 0).UTC()
		year, month, day, weekday := civilFromDays(days)
		if year != int64(want.Year()) || month != int(want.Month()) || day != want.Day() || weekday != int(want.Weekday()) {
			t.Fatalf("%d: got %d-%d-%d (%d), want %s", days, year, month, day, weekday, want.Format("2006-01-02 Mon"))
		}
		if back := daysFromCivil(year, int64(month), int64(day)); back != days {
			t.Fatalf("%d: back to %d", days, back)
		}
	}
}

// parseTime finds the right offset on both sides of a change of daylight
// saving time
func TestParseTimeZone(t *testing.T) {
	zone, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip(err)
	}
	previous := time.Local
	time.Local = zone
	defer func() { time.Local = previous }()

	for _, text := range []string{"2026-03-08 01:30", "2026-03-08 03:30", "2026-11-01 00:30", "2026-07-01 12:00"} {
		want, _ := time.ParseInLocation("2006-01-02 15:04", text, zone)
		got, ok := readTime(text, "YYYY-MM-DD HH:mm")
		if !ok || got != want.Unix() {
			t.Errorf("%s: got %d, want %d", text, got, want.Unix())
		}
		if back := formatTime(got, "YYYY-MM-DD HH:mm"); back != text {
			t.Errorf("%s: formats as %s", text, back)
		}
	}
}
