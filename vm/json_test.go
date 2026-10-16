package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`parseJSON("{\"b\": 1, \"a\": [true, null, 2.5, \"x\"]}")`, `{"b": 1, "a": [true, null, 2.5, "x"]}`},
		{`parseJSON("{\"b\": 1, \"a\": 2}").keys()`, `["b", "a"]`},
		{`parseJSON("12345678901234567890")`, "1.2345678901234567e+19"},
		{`parseJSON("1e2")`, "100"},
		{`type(parseJSON("1e2"))`, "float"},
		{`parseJSON("  \"héllo\\n\"  ")`, "héllo\n"},
		{`parseJSON("[1, 2")`, "ERROR: invalid JSON at line 1, column 6: the text ended too early"},
		{`parseJSON("{\n  \"a\": x\n}")`, "ERROR: invalid JSON at line 2, column 8: unexpected 'x'"},
		{`parseJSON("{\"a\" 1}")`, "ERROR: invalid JSON at line 1, column 6: unexpected '1', expected ':'"},
		{`parseJSON("[1 2]")`, "ERROR: invalid JSON at line 1, column 4: unexpected '2', expected ',' or ']'"},
		{`parseJSON("\"\\ud83d\\ude00 \\u00e9\\t\"")`, "😀 é\t"},
		{`parseJSON("[] ")`, "[]"},
		{`parseJSON("-0.5e1")`, "-5"},
		{`parseJSON("01")`, "ERROR: invalid JSON at line 1, column 2: unexpected text after the value"},
		{`toJSON({"empty": [], "o": {}}, 2)`, "{\n  \"empty\": [],\n  \"o\": {}\n}"},
		{`toJSON("tab\t\"q\" \\ ü")`, `"tab\t\"q\" \\ ü"`},
		{`parseJSON("1 2")`, "ERROR: invalid JSON at line 1, column 3: unexpected text after the value"},
		{`parseJSON(5)`, "ERROR: argument to parseJSON must be a string, got an integer"},
		{`toJSON({"name": "Ana", "tags": ["a", "<b>"], "age": 30, "ok": true, "none": null, "pi": 3.5})`,
			`{"name":"Ana","tags":["a","<b>"],"age":30,"ok":true,"none":null,"pi":3.5}`},
		{`toJSON([1, {"a": 2}], 2)`, "[\n  1,\n  {\n    \"a\": 2\n  }\n]"},
		{`toJSON({1: "x"})`, `{"1":"x"}`},
		{`toJSON(1.0 / 0)`, "ERROR: cannot convert Inf to JSON"},
		{`toJSON([fn() { 1 }])`, "ERROR: cannot convert a function to JSON"},
		{`let data = {"items": [1, 2, 3], "label": "x"}` + "\nparseJSON(toJSON(data)) == data", "true"},
		{`let data = {"items": [1, 2, 3], "label": "x"}` + "\ntoJSON(parseJSON(toJSON(data))) == toJSON(data)", "true"},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		got := result.Inspect()
		if err, ok := result.(*object.Error); ok {
			got = "ERROR: " + err.Message
		}
		if got != tt.expected {
			t.Errorf("%s\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
