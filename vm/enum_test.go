package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

const statusEnum = "enum Status { Active, Blocked\n Pending }\n"

// An enum's values are equal only to themselves and match by name
func TestEnum(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"Status.Active", "Status.Active"},
		{"[type(Status.Active), type(Status), Status.Blocked.name]", `["Status", "enum", "Blocked"]`},
		{"[Status.Active == Status.Active, Status.Active == Status.Blocked, Status.Active == \"Active\"]", "[true, false, false]"},
		{"Status.values()", "[Status.Active, Status.Blocked, Status.Pending]"},
		{"let e = Status\ne.Pending", "Status.Pending"},
		{"toJSON([Status.Blocked])", `["Blocked"]`},
		{"Status.values().map(s => match s { Status.Active => 1, Status.Blocked, Status.Pending => 0 })", "[1, 0, 0]"},
		{"match Status.Pending { Status.Active => 1, _ => 2 }", "2"},
		{"fn f(Status) { Status.x }\nf({x: 3})", "3"},
	}
	for _, tt := range tests {
		if got := testEval(statusEnum + tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestEnumErrors(t *testing.T) {
	tests := []struct{ input, message string }{
		{"Status.Done", "enum Status has no Done; its values are Active, Blocked, Pending"},
		{"let e = Status\ne.Done", "enum Status has no Done"},
		{"match Status.Pending { Status.Active => 1 }", "match has no arm for Blocked, Pending of enum Status"},
		{"match Status.Blocked { Status.Active => 1, Status.Blocked if false => 2, Status.Pending => 3 }", "match has no arm for Blocked of enum Status"},
		{"enum Status { A }", "identifier already declared: Status"},
		{"Status = 1", "cannot reassign constant: Status"},
	}
	for _, tt := range tests {
		result, ok := testEval(statusEnum + tt.input).(*object.Error)
		if !ok || !strings.Contains(result.Message, tt.message) {
			t.Errorf("%s: got %v, want %q", tt.input, result, tt.message)
		}
	}
	// A match missing a value is only an error when it gets there
	if got := testEval(statusEnum + "match Status.Active { Status.Active => 1 }").Inspect(); got != "1" {
		t.Errorf("got %s", got)
	}
}
