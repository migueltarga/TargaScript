package repl

import (
	"regexp"

	"github.com/migueltarga/TargaScript/evaluator"
)

// goRegex runs string patterns with Go's regexp package.
type goRegex struct{}

func (goRegex) Compile(pattern string) (evaluator.Regex, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return goPattern{re}, nil
}

type goPattern struct{ re *regexp.Regexp }

func (p goPattern) MatchString(s string) bool            { return p.re.MatchString(s) }
func (p goPattern) FindStringSubmatch(s string) []string { return p.re.FindStringSubmatch(s) }
func (p goPattern) FindAllStringSubmatch(s string) [][]string {
	return p.re.FindAllStringSubmatch(s, -1)
}
func (p goPattern) ReplaceAllString(s, r string) string { return p.re.ReplaceAllString(s, r) }
func (p goPattern) Split(s string) []string             { return p.re.Split(s, -1) }
