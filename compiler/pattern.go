package compiler

import (
	"strings"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
	"github.com/migueltarga/TargaScript/token"
)

// matchPattern is the pattern of a match arm that takes the subject apart,
// being compiled: the constant for the VM, the expressions it compares
// with, and the names it binds, in order.
type matchPattern struct {
	root   *object.Pattern
	values []ast.Expression
	names  []*ast.Identifier
}

// isStructural reports whether the pattern of a match arm takes the
// subject apart: an array or object literal, whose names bind.
func isStructural(pattern ast.Expression) bool {
	switch pattern.(type) {
	case *ast.ArrayLiteral, *ast.HashLiteral:
		return true
	}
	return false
}

// buildPattern turns the array or object literal of a match arm into a
// pattern. It emits an error and returns nil for one that can't be a
// pattern.
func (c *Compiler) buildPattern(expr ast.Expression) *matchPattern {
	mp := &matchPattern{}
	whole := expr.String()
	tok := tokenOf(expr)
	root := c.buildPart(mp, expr, tok, whole)
	if root == nil {
		return nil
	}
	root.Values, root.Binds, root.Source = len(mp.values), len(mp.names), whole
	mp.root = root
	return mp
}

// buildPart builds one part of a pattern: a name binds, _ matches anything,
// a literal array or object takes the value apart, a range matches what's
// between its ends, and any other expression must be equal to the value.
func (c *Compiler) buildPart(mp *matchPattern, expr ast.Expression, tok token.Token, whole string) *object.Pattern {
	switch node := expr.(type) {
	case *ast.Identifier:
		if node.Value == "_" {
			return &object.Pattern{Kind: object.PatternAny}
		}
		for _, name := range mp.names {
			if name.Value == node.Value {
				c.emitError(tok, "%s is bound twice in %s", node.Value, whole)
				return nil
			}
		}
		mp.names = append(mp.names, node)
		return &object.Pattern{Kind: object.PatternBind, Index: len(mp.names) - 1}

	case *ast.ArrayLiteral:
		pattern := &object.Pattern{Kind: object.PatternArray}
		for i, element := range node.Elements {
			if spread, ok := element.(*ast.SpreadExpression); ok {
				if i != len(node.Elements)-1 {
					c.emitError(tok, "%s must be the last element in %s", spread.String(), whole)
					return nil
				}
				if pattern.Rest = c.buildRest(mp, spread, tok, whole); pattern.Rest == nil {
					return nil
				}
				break
			}
			part := c.buildPart(mp, element, tok, whole)
			if part == nil {
				return nil
			}
			pattern.Parts = append(pattern.Parts, part)
		}
		return pattern

	case *ast.HashLiteral:
		pattern := &object.Pattern{Kind: object.PatternObject}
		for _, name := range node.Order {
			pair := node.Pairs[name]
			if spread, ok := pair.Key.(*ast.SpreadExpression); ok {
				if pattern.Rest = c.buildRest(mp, spread, tok, whole); pattern.Rest == nil {
					return nil
				}
				continue
			}
			if parser.PatternOnly(pair) {
				c.emitError(tok, "a match pattern can't give a default, %s takes the property as it is", whole)
				return nil
			}
			var key string
			switch k := pair.Key.(type) {
			case *ast.Identifier:
				key = k.Value
			case *ast.StringLiteral:
				key = k.Value
			default:
				c.emitError(tok, "a match pattern names the properties it takes, and %s isn't a name in %s", pair.Key.String(), whole)
				return nil
			}
			part := c.buildPart(mp, pair.Value, tok, whole)
			if part == nil {
				return nil
			}
			pattern.Keys = append(pattern.Keys, &object.String{Value: key})
			pattern.Parts = append(pattern.Parts, part)
		}
		return pattern

	case *ast.AssignmentExpression:
		if node.Operator == "" {
			c.emitError(tok, "a match pattern can't give a default, %s takes the element as it is", whole)
			return nil
		}

	case *ast.InfixExpression:
		if node.Operator == ".." {
			mp.values = append(mp.values, node.Left, node.Right)
			return &object.Pattern{Kind: object.PatternRange, Index: len(mp.values) - 2}
		}
	}

	mp.values = append(mp.values, expr)
	return &object.Pattern{Kind: object.PatternValue, Index: len(mp.values) - 1}
}

// buildRest builds the part that takes the rest, from `...rest` or `..._`.
func (c *Compiler) buildRest(mp *matchPattern, spread *ast.SpreadExpression, tok token.Token, whole string) *object.Pattern {
	if _, ok := spread.Value.(*ast.Identifier); !ok {
		c.emitError(tok, "... takes a name in a pattern, like [first, ...rest], not %s", spread.Value.String())
		return nil
	}
	return c.buildPart(mp, spread.Value, tok, whole)
}

// nameList prints the names a pattern binds, for errors.
func nameList(names []*ast.Identifier) string {
	if len(names) == 0 {
		return "nothing"
	}
	list := []string{}
	for _, name := range names {
		list = append(list, name.Value)
	}
	return strings.Join(list, ", ")
}

// tokenOf returns the token an expression starts with.
func tokenOf(expr ast.Expression) token.Token {
	switch node := expr.(type) {
	case *ast.ArrayLiteral:
		return node.Token
	case *ast.HashLiteral:
		return node.Token
	}
	return token.Token{}
}
