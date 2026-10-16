package format

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSource(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"indents blocks", "fn f() {\nif x {\nprint(1)\n}\n}\n", "fn f() {\n    if x {\n        print(1)\n    }\n}\n"},
		{"fixes wrong indentation", "  let a = 1\n        let b = 2\n", "let a = 1\nlet b = 2\n"},
		{"brackets and parens", "let a = [\n1,\n2,\n]\nprint(\nf(1)\n)\n", "let a = [\n    1,\n    2,\n]\nprint(\n    f(1)\n)\n"},
		{"continuation lines", "let x = list\n.map(f)\n|> g\n", "let x = list\n    .map(f)\n    |> g\n"},
		{"trailing spaces and blank lines", "\n\nlet a = 1   \n\n\n\nlet b = 2\n\n\n", "let a = 1\n\nlet b = 2\n"},
		{"strings keep their lines", "let s = `one\n  two   \n    three`\nlet t = \"a\n   b\"\n", "let s = `one\n  two   \n    three`\nlet t = \"a\n   b\"\n"},
		{"braces in strings and comments don't count", "fn f() {\nlet s = \"{\"\n// }\n/* { */\nprint(s)\n}\n", "fn f() {\n    let s = \"{\"\n    // }\n    /* { */\n    print(s)\n}\n"},
		{"block comments keep their lines", "/*\n   art\n */\nlet a = 1\n", "/*\n   art\n */\nlet a = 1\n"},
		{"interpolations with braces", "fn f() {\nprint(\"${ {a: 1}.a } }\")\nprint(1)\n}\n", "fn f() {\n    print(\"${ {a: 1}.a } }\")\n    print(1)\n}\n"},
		{"strings inside interpolations", "fn f() {\nprint(\"a ${\"}\"} b\")\n}\n", "fn f() {\n    print(\"a ${\"}\"} b\")\n}\n"},
		{"spread isn't a continuation", "let a = [\n...b\n]\n", "let a = [\n    ...b\n]\n"},
		{"tabs", "if x {\n\tprint(1)\n}\n", "if x {\n    print(1)\n}\n"},
		{"several brackets on a line are one level", "test(\"x\", fn() {\nassert(true)\n})\n", "test(\"x\", fn() {\n    assert(true)\n})\n"},
		{"a block in a continuation", "let x = list\n.map(n => {\nn * 2\n})\n", "let x = list\n    .map(n => {\n        n * 2\n    })\n"},
		{"objects in calls", "f({\na: 1,\nb: [\n2,\n],\n})\n", "f({\n    a: 1,\n    b: [\n        2,\n    ],\n})\n"},
		{"empty", "\n\n", ""},
	}
	for _, tt := range tests {
		if got := Source(tt.in); got != tt.want {
			t.Errorf("%s:\ngot:\n%s\nwant:\n%s", tt.name, got, tt.want)
		}
		if again := Source(tt.want); again != tt.want {
			t.Errorf("%s: formatting the result changes it:\n%s", tt.name, again)
		}
	}
}

// Every example and test in the repository formats to something that
// formats to itself.
func TestFormattingIsStable(t *testing.T) {
	files, _ := filepath.Glob("../examples/*.tg")
	more, _ := filepath.Glob("../tests/*.tg")
	tour, _ := filepath.Glob("../playground/tour/*.tg")
	for _, file := range append(append(files, more...), tour...) {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		once := Source(string(source))
		if twice := Source(once); twice != once {
			t.Errorf("%s: formatting twice gives a different result", file)
		}
	}
}
