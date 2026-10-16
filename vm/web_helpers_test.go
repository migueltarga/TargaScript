package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestWebHelpers(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`escapeHTML("<b>Tom & \"Jerry\"</b> 'x'")`, "&lt;b&gt;Tom &amp; &#34;Jerry&#34;&lt;/b&gt; &#39;x&#39;"},
		{`escapeHTML(42)`, "42"},
		{`escapeHTML("plain")`, "plain"},
		{`encodeURL("a b&c=d/é")`, "a%20b%26c%3Dd%2F%C3%A9"},
		{`encodeURL("safe-_.~AZaz09")`, "safe-_.~AZaz09"},
		{`decodeURL("a%20b%26c%3Dd%2F%C3%A9")`, "a b&c=d/é"},
		{`decodeURL("a+b")`, "a b"},
		{`decodeURL(encodeURL("100% sure?"))`, "100% sure?"},
	}
	for _, tt := range tests {
		str, ok := testEval(tt.input).(*object.String)
		if !ok || str.Value != tt.expected {
			t.Errorf("%s: got %v, want %q", tt.input, testEval(tt.input), tt.expected)
		}
	}

	for _, input := range []string{`decodeURL("%")`, `decodeURL("%zz")`, `decodeURL("%4")`} {
		err, ok := testEval(input).(*object.Error)
		if !ok || !strings.Contains(err.Message, "not followed by two hex digits") {
			t.Errorf("%s: got %v", input, testEval(input))
		}
	}
}
