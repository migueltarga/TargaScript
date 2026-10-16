package repl

import "testing"

func TestCheckBrowserGlobals(t *testing.T) {
	source := `let button = js.document.getElementById("run")
print(targascript.version, nope)`
	problems := Check(source, "")
	if len(problems) != 1 || problems[0].Message != `"nope" is not defined` {
		t.Errorf("got %v, want only nope undefined", problems)
	}
}
