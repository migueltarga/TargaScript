package compiler

import (
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
	names  []string
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
	root := c.buildPart(mp, expr, tokenOf(expr))
	if root == nil {
		return nil
	}
	root.Values, root.Binds = len(mp.values), len(mp.names)
	mp.root = root
	return mp
}

// buildPart builds one part of a pattern: a name binds, _ matches anything,
// a literal array or object takes the value apart, a range matches what's
// between its ends, and any other expression must be equal to the value.
func (c *Compiler) buildPart(mp *matchPattern, expr ast.Expression, tok token.Token) *object.Pattern {
	switch node := expr.(type) {
	case *ast.Identifier:
		if node.Value == "_" {
			return &object.Pattern{Kind: object.PatternAny}
		}
		for _, name := range mp.names {
			if name == node.Value {
				c.emitError(tok, "%s is bound twice in the pattern", name)
				return nil
			}
		}
		mp.names = append(mp.names, node.Value)
		return &object.Pattern{Kind: object.PatternBind, Index: len(mp.names) - 1}

	case *ast.ArrayLiteral:
		pattern := &object.Pattern{Kind: object.PatternArray}
		for i, element := range node.Elements {
			if spread, ok := element.(*ast.SpreadExpression); ok {
				if _, isName := spread.Value.(*ast.Identifier); !isName || i != len(node.Elements)-1 {
					c.emitError(tok, "... in a pattern takes a name, at the end, like [first, ...rest]")
					return nil
				}
				pattern.Rest = c.buildPart(mp, spread.Value, tok)
				if pattern.Rest == nil {
					return nil
				}
				break
			}
			part := c.buildPart(mp, element, tok)
			if part == nil {
				return nil
			}
			pattern.Parts = append(pattern.Parts, part)
		}
		return pattern

	case *ast.HashLiteral:
		pattern := &object.Pattern{Kind: object.PatternObject}
		keys := &object.Array{}
		for _, name := range node.Order {
			pair := node.Pairs[name]
			if spread, ok := pair.Key.(*ast.SpreadExpression); ok {
				if _, isName := spread.Value.(*ast.Identifier); !isName {
					c.emitError(tok, "... in a pattern takes a name, like {name, ...rest}")
					return nil
				}
				pattern.Rest = c.buildPart(mp, spread.Value, tok)
				if pattern.Rest == nil {
					return nil
				}
				continue
			}
			if parser.PatternOnly(pair) {
				c.emitError(tok, "a match pattern can't give a default, it takes the value as it is")
				return nil
			}
			var key string
			switch k := pair.Key.(type) {
			case *ast.Identifier:
				key = k.Value
			case *ast.StringLiteral:
				key = k.Value
			default:
				c.emitError(tok, "a match pattern names the properties it takes, like {name, age}")
				return nil
			}
			part := c.buildPart(mp, pair.Value, tok)
			if part == nil {
				return nil
			}
			keys.Elements = append(keys.Elements, &object.String{Value: key})
			pattern.Parts = append(pattern.Parts, part)
		}
		pattern.Keys = keys
		return pattern

	case *ast.AssignmentExpression:
		if node.Operator == "" {
			c.emitError(tok, "a match pattern can't give a default, it takes the value as it is")
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
