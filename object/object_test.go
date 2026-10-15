package object

import "testing"

func TestInspectQuotesStringsInCollections(t *testing.T) {
	str := func(s string) *String { return &String{Value: s} }

	hash := NewHash()
	hash.Set(str("name"), str("Ana"))
	hash.Set(&Integer{Value: 1}, &Array{Elements: []Object{str("x")}})

	tests := []struct {
		obj      Object
		expected string
	}{
		{str("plain"), "plain"},
		{&Array{Elements: []Object{str("a"), &Integer{Value: 1}, str("1")}}, `["a", 1, "1"]`},
		{&Array{Elements: []Object{str("line\nbreak"), str(`say "hi"`), str("olá")}}, `["line\nbreak", "say \"hi\"", "olá"]`},
		{&Array{Elements: []Object{&Array{Elements: []Object{str("nested")}}}}, `[["nested"]]`},
		{hash, `{"name": "Ana", 1: ["x"]}`},
		{&Array{Elements: []Object{}}, "[]"},
	}

	for _, tt := range tests {
		if got := tt.obj.Inspect(); got != tt.expected {
			t.Errorf("got=%s, want=%s", got, tt.expected)
		}
	}
}
