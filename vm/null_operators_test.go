package vm

import "testing"

func TestNullOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"null ?? 5", "5"},
		{"0 ?? 5", "0"},
		{"false ?? 5", "false"},
		{`"" ?? "default"`, ""},
		{"null ?? null ?? 3", "3"},
		{`let config = {"port": 80}` + "\nconfig.host ?? \"localhost\"", "localhost"},
		{"1 ?? missing", "1"},
		{"null || false ?? 7", "false"},
		{`let user = null` + "\nuser?.name", "null"},
		{`let user = {"name": "Ana"}` + "\nuser?.name", "Ana"},
		{`let user = null` + "\nuser?.name.upper()", "null"},
		{`let user = {"name": "ana"}` + "\nuser?.name.upper()", "ANA"},
		{`let user = null` + "\nuser?.greet(missing)", "null"},
		{`let user = {"address": null}` + "\nuser.address?.city ?? \"unknown\"", "unknown"},
		{`let user = {"address": {"city": "Rio"}}` + "\nuser.address?.city ?? \"unknown\"", "Rio"},
		{`let list = null` + "\n[1, list?.length, 3]", "[1, null, 3]"},
		{`let list = null` + "\n[list?.length, \"x\"]", `[null, "x"]`},
		{`let a = null` + "\nlet b = {\"c\": [10, 20]}\n[a?.c[1], b?.c[1]]", "[null, 20]"},
		{"let x = 5\nx?.foo", `ERROR at line 2, column 2: cannot read property "foo" of an integer`},
		{`let user = {}` + "\nuser.profile?.name", "null"},
		{`fn find(id) { if id == 1 { {"name": "Ana"} } }` + "\n[find(1)?.name, find(2)?.name ?? \"none\"]", `["Ana", "none"]`},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
