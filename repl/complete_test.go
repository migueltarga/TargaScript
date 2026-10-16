package repl

import "testing"

func TestComplete(t *testing.T) {
	names := []string{"parseJSON", "parseTime", "print", "printf", "price", "total"}
	tests := []struct{ prefix, want string }{
		{"tot", "al"},
		{"parseJ", "SON"},
		{"pars", "e"},
		{"pri", ""},
		{"prin", "t"},
		{"total", ""},
		{"zzz", ""},
	}
	for _, tt := range tests {
		if got := complete(tt.prefix, names); got != tt.want {
			t.Errorf("complete(%q) = %q, want %q", tt.prefix, got, tt.want)
		}
	}
}
