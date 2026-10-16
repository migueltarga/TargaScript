package main

import (
	"bytes"
	"os"
	"regexp"
	"testing"

	"github.com/migueltarga/TargaScript/repl"
)

func TestLinkRoundTrip(t *testing.T) {
	code := "print(\"hello\")\nlet s = `${1 + 1}`\n"
	got, err := Code(Link(code))
	if err != nil || got != code {
		t.Errorf("got %q, %v", got, err)
	}
}

// Every playground link in the README and the docs opens code that runs
// without a problem, so the links can't fall behind the language.
func TestReadmeLinks(t *testing.T) {
	links := []string{}
	for _, file := range []string{"../../README.md", "../../docs/COOKBOOK.md"} {
		text, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		found := regexp.MustCompile(`https://migueltarga\.github\.io/TargaScript/#code=[A-Za-z0-9_-]+`).FindAllString(string(text), -1)
		if len(found) == 0 {
			t.Fatalf("no playground links in %s", file)
		}
		links = append(links, found...)
	}
	for _, link := range links {
		code, err := Code(link)
		if err != nil {
			t.Errorf("%s: %v", link[:60], err)
			continue
		}
		if problems := repl.Check(code, ""); len(problems) > 0 {
			t.Errorf("a README link opens code with problems: %v\n%s", problems[0].Message, code)
			continue
		}
		// Code that doesn't reach the network should also run
		if !bytes.Contains([]byte(code), []byte("fetch(")) {
			var out bytes.Buffer
			if err := repl.EvalSource(code, &out, true, false, false); err != nil {
				t.Errorf("a README link opens code that fails: %v\n%s", err, code)
			}
		}
	}
}
