package main

import "testing"

func TestRunCheck(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{"fine", map[string]string{"a.tg": "let x = 1\nprint(x)", "lib/b.tg": "fn f() { 1 }"}, 0},
		{"syntax error", map[string]string{"a.tg": "let x = "}, 1},
		{"undefined name", map[string]string{"a.tg": "print(nope)"}, 1},
		{"a load that works", map[string]string{"a.tg": "load \"lib/b.tg\"\nf()", "lib/b.tg": "fn f() { 1 }"}, 0},
		{"a load of a missing file", map[string]string{"a.tg": "load \"nope.tg\""}, 1},
		{"hidden directories are skipped", map[string]string{"a.tg": "1", ".git/x.tg": "let ="}, 0},
	}
	for _, tt := range tests {
		if got := runCheck([]string{writeFiles(t, tt.files)}); got != tt.want {
			t.Errorf("%s: exit code %d, want %d", tt.name, got, tt.want)
		}
	}
}
