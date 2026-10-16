package vm

import "testing"

func TestBareReturn(t *testing.T) {
	tests := []string{
		"fn f() { return }\nf()",
		"fn f() { return; }\nf()",
		"fn f() {\n  return\n}\nf()",
		"fn f(x) {\n  if x { return }\n  return 1\n}\nf(true)",
		"return",
	}

	for _, input := range tests {
		evaluated := testEval(input)
		if evaluated != NULL {
			t.Errorf("%q: expected null, got %s", input, evaluated.Inspect())
		}
	}

	testIntegerObject(t, testEval("fn f(x) {\n  if x { return }\n  return 1\n}\nf(false)"), 1, "bare return in branch not taken")
	testIntegerObject(t, testEval("fn f() { return 5 }\nf()"), 5, "return with value")
	testIntegerObject(t, testEval("fn f() {\n  repeat i in 10 {\n    if i == 3 { return }\n  }\n  return 0\n}\nf()\n7"), 7, "bare return inside loop")
}
