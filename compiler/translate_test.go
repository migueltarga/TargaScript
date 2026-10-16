package compiler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/parser"
)

// Every program the repository has translates to register code
func TestTranslatePrograms(t *testing.T) {
	var files []string
	for _, pattern := range []string{"../examples/*.tg", "../tests/*.tg", "../bench/*.tg", "../playground/*.tg", "../playground/tour/*.tg"} {
		matches, _ := filepath.Glob(pattern)
		files = append(files, matches...)
	}
	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		p := parser.New(lexer.New(string(source)))
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			continue
		}
		c := New()
		c.Dir = filepath.Dir(file)
		if _, err := c.Compile(program); err != nil {
			t.Errorf("%s: %v", file, err)
		}
	}
}
