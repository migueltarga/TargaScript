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
