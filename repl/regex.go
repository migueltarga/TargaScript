package repl

import (
	"regexp"
	"strings"

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

func (p goPattern) ReplaceAllFunc(s string, replace func([]string) string) string {
	var out strings.Builder
	last := 0
	for _, m := range p.re.FindAllStringSubmatchIndex(s, -1) {
		out.WriteString(s[last:m[0]])
		groups := make([]string, len(m)/2)
		for i := range groups {
			if m[2*i] >= 0 {
				groups[i] = s[m[2*i]:m[2*i+1]]
			}
		}
		out.WriteString(replace(groups))
		last = m[1]
	}
	out.WriteString(s[last:])
	return out.String()
}
