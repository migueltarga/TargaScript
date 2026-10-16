package evaluator

import (
	"testing"
	"time"

	"github.com/migueltarga/TargaScript/object"
)

func TestDateLayout(t *testing.T) {
	stamp := time.Date(2026, time.March, 5, 14, 7, 9, 0, time.UTC)
	tests := []struct{ format, want string }{
		{"YYYY-MM-DD", "2026-03-05"},
		{"DD/MM/YY HH:mm:ss", "05/03/26 14:07:09"},
		{"hh:mm A", "02:07 PM"},
		{"dddd, MMMM DD", "Thursday, March 05"},
		{"ddd MMM", "Thu Mar"},
		{"2006-01-02 15:04", "2026-03-05 14:07"},
	}
	for _, tt := range tests {
		if got := stamp.Format(dateLayout(tt.format)); got != tt.want {
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
	previous := time.Local
	time.Local = time.UTC
	defer func() { time.Local = previous }()

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
	formatted := time.Unix(stamp, 0).Format(dateLayout("YYYY-MM-DD HH:mm:ss"))
	if n := parseTimeBuiltin(&object.String{Value: formatted}, &object.String{Value: "YYYY-MM-DD HH:mm:ss"}).(*object.Integer); n.Value != stamp {
		t.Errorf("round trip: got %d, want %d", n.Value, stamp)
	}
}
