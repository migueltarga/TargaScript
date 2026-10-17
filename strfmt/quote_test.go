package strfmt

import (
	"strconv"
	"testing"
	"unicode/utf8"
)

// Quoting gives what strconv gives for every character that both count as
// printing or both don't, and only escapes to ASCII the same way
func TestQuoteLikeStrconv(t *testing.T) {
	differ := 0
	for r := rune(0); r <= utf8.MaxRune; r++ {
		s := string(r)
		if got, want := quote(s, '"', true), strconv.QuoteToASCII(s); got != want {
			t.Fatalf("ASCII %U: got %s, want %s", r, got, want)
		}
		if printable(r) != strconv.IsPrint(r) {
			differ++
			continue
		}
		if got, want := quote(s, '"', false), strconv.Quote(s); got != want {
			t.Fatalf("%U: got %s, want %s", r, got, want)
		}
		if got, want := quote(s, '\'', false), strconv.QuoteRune(r); got != want {
			t.Fatalf("rune %U: got %s, want %s", r, got, want)
		}
	}
	t.Logf("%d characters print by one and not the other", differ)
}

func FuzzQuote(f *testing.F) {
	f.Add("say \"hi\"\n\x00\xff`é")
	f.Fuzz(func(t *testing.T, s string) {
		if got, want := quote(s, '"', true), strconv.QuoteToASCII(s); got != want {
			t.Errorf("%q: got %s, want %s", s, got, want)
		}
		if got, want := canBackquote(s), strconv.CanBackquote(s); got != want {
			t.Errorf("%q: canBackquote %v, want %v", s, got, want)
		}
	})
}
