package ast

import (
	"bytes"
	"strings"

	"github.com/migueltarga/TargaScript/strfmt"
	"github.com/migueltarga/TargaScript/token"
)

type Node interface {
	TokenLiteral() string
	String() string
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	} else {
		return ""
	}
}

func (p *Program) String() string {
	var out bytes.Buffer

	for i, s := range p.Statements {
		out.WriteString(s.String())

		if i < len(p.Statements)-1 {
			out.WriteString("\n")
		}
	}

	return out.String()
}

type Identifier struct {
	Token token.Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

type LetStatement struct {
	Token token.Token
	Name  *Identifier
	Value Expression

	// Pattern replaces Name in `let [a, b] = ...` and `let {a, b} = ...`
	Pattern *Pattern
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LetStatement) String() string {
	var out bytes.Buffer

	out.WriteString(ls.TokenLiteral() + " ")
	out.WriteString(bindingString(ls.Name, ls.Pattern))
	out.WriteString(" = ")

	if ls.Value != nil {
		out.WriteString(nodeString(ls.Value))
	}

	return out.String()
}

type ConstStatement struct {
	Token token.Token
	Name  *Identifier
	Value Expression

	// Pattern replaces Name in `const [a, b] = ...` and `const {a, b} = ...`
	Pattern *Pattern
}

func (cs *ConstStatement) statementNode()       {}
func (cs *ConstStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ConstStatement) String() string {
	var out bytes.Buffer

	out.WriteString(cs.TokenLiteral() + " ")
	out.WriteString(bindingString(cs.Name, cs.Pattern))
	out.WriteString(" = ")

	if cs.Value != nil {
		out.WriteString(nodeString(cs.Value))
	}

	return out.String()
}

type ReturnStatement struct {
	Token       token.Token
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	var out bytes.Buffer

	out.WriteString(rs.TokenLiteral() + " ")

	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}

	return out.String()
}

type ExpressionStatement struct {
	Token      token.Token
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

type IntegerLiteral struct {
	Token token.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

type FloatLiteral struct {
	Token token.Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

type PrefixExpression struct {
	Token    token.Token
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(nodeString(pe.Right))
	out.WriteString(")")

	return out.String()
}

// nodeString is the String of a node, or "" for one missing because of a
// syntax error, so error messages can show a partly parsed expression.
func nodeString(node Node) (s string) {
	if node == nil {
		return ""
	}
	// A nil pointer in the interface panics on its fields
	defer func() {
		if recover() != nil {
			s = ""
		}
	}()
	return node.String()
}

type InfixExpression struct {
	Token    token.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(nodeString(ie.Left))
	out.WriteString(" " + ie.Operator + " ")
	out.WriteString(nodeString(ie.Right))
	out.WriteString(")")

	return out.String()
}

type PostfixExpression struct {
	Token    token.Token
	Left     Expression
	Operator string
}

func (pe *PostfixExpression) expressionNode()      {}
func (pe *PostfixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PostfixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(nodeString(pe.Left))
	out.WriteString(pe.Operator)
	out.WriteString(")")

	return out.String()
}

type Boolean struct {
	Token token.Token
	Value bool
}

func (b *Boolean) expressionNode()      {}
func (b *Boolean) TokenLiteral() string { return b.Token.Literal }
func (b *Boolean) String() string       { return b.Token.Literal }

type NullLiteral struct {
	Token token.Token
}

func (n *NullLiteral) expressionNode()      {}
func (n *NullLiteral) TokenLiteral() string { return n.Token.Literal }
func (n *NullLiteral) String() string       { return n.Token.Literal }

type IfExpression struct {
	Token       token.Token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IfExpression) String() string {
	var out bytes.Buffer

	out.WriteString("if")
	out.WriteString("(")
	out.WriteString(ie.Condition.String())
	out.WriteString(" ")
	out.WriteString(ie.Consequence.String())

	if ie.Alternative != nil {
		out.WriteString("else ")
		out.WriteString(ie.Alternative.String())
	}

	return out.String()
}

type BlockStatement struct {
	Token      token.Token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out bytes.Buffer

	out.WriteString("{ ")
	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}
	out.WriteString(" }")

	return out.String()
}

type FunctionLiteral struct {
	Token      token.Token
	Name       *Identifier
	Parameters []*Identifier
	Defaults   []Expression // default value of each parameter, or nil
	Rest       bool         // the last parameter collects the extra arguments
	Patterns   []*Pattern   // a pattern taking a parameter apart, or nil
	Body       *BlockStatement

	// Arrow is set for functions written x => body
	Arrow bool

	// ImplicitIt is set for an argument that uses it without defining it,
	// as in xs.map(it * 2): a function of it, unless it is a variable
	ImplicitIt bool
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) String() string {
	var out bytes.Buffer

	params := ParameterStrings(fl.Parameters, fl.Defaults, fl.Rest)

	if fl.ImplicitIt {
		return fl.Body.String()
	}
	if fl.Arrow {
		return "(" + strings.Join(params, ", ") + ") => " + fl.Body.String()
	}

	out.WriteString(fl.TokenLiteral())

	if fl.Name != nil {
		out.WriteString(" ")
		out.WriteString(fl.Name.String())
	}

	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") ")
	out.WriteString(fl.Body.String())

	return out.String()
}

type CallExpression struct {
	Token     token.Token
	Function  Expression
	Arguments []Expression

	// Piped is set for value |> f(args), with value as the first
	// argument. When f isn't a function the program has, the call is the
	// method f of value: xs |> map(it * 2) is xs.map(it * 2).
	Piped bool
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var out bytes.Buffer

	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}

	out.WriteString(nodeString(ce.Function))
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")

	return out.String()
}

// TemplateLiteral is a string with ${...} interpolations. Parts alternate
// between *StringLiteral text and the interpolated expressions.
type TemplateLiteral struct {
	Token token.Token
	Parts []Expression
}

func (tl *TemplateLiteral) expressionNode()      {}
func (tl *TemplateLiteral) TokenLiteral() string { return tl.Token.Literal }
func (tl *TemplateLiteral) String() string {
	var out bytes.Buffer

	out.WriteString("\"")
	for _, part := range tl.Parts {
		if text, ok := part.(*StringLiteral); ok {
			out.WriteString(text.Value)
		} else {
			out.WriteString("${" + part.String() + "}")
		}
	}
	out.WriteString("\"")

	return out.String()
}

type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return "\"" + sl.Value + "\"" }

type ArrayLiteral struct {
	Token    token.Token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	var out bytes.Buffer

	elements := []string{}
	for _, el := range al.Elements {
		elements = append(elements, el.String())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}

type IndexExpression struct {
	Token token.Token
	Left  Expression
	Index Expression

	// Optional is set for list?.[i], which is null when list is null
	Optional bool
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(nodeString(ie.Left))
	if ie.Optional {
		out.WriteString("?.")
	}
	out.WriteString("[")
	out.WriteString(nodeString(ie.Index))
	out.WriteString("])")

	return out.String()
}

type LoadStatement struct {
	Token token.Token
	Path  *StringLiteral
	Alias *Identifier
}

func (ls *LoadStatement) statementNode()       {}
func (ls *LoadStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LoadStatement) String() string {
	var out bytes.Buffer

	out.WriteString(ls.TokenLiteral() + " ")
	out.WriteString(ls.Path.String())

	if ls.Alias != nil {
		out.WriteString(" as ")
		out.WriteString(ls.Alias.String())
	}

	return out.String()
}

type RepeatStatement struct {
	Token      token.Token
	Iterator   *Identifier
	Collection Expression
	Body       *BlockStatement

	// Key is set in `repeat key, value in collection`, where Iterator is
	// the value: the index for arrays, strings and ranges, the key for
	// objects
	Key *Identifier

	// Pattern takes each value apart, as in `repeat [x, y] in points`
	Pattern *Pattern
}

func (rs *RepeatStatement) statementNode()       {}
func (rs *RepeatStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *RepeatStatement) String() string {
	var out bytes.Buffer

	out.WriteString(rs.TokenLiteral() + " ")

	if rs.Key != nil {
		out.WriteString(nodeString(rs.Key))
		out.WriteString(", ")
	}
	if rs.Iterator != nil {
		out.WriteString(rs.Iterator.String())
		out.WriteString(" in ")
	}

	out.WriteString(rs.Collection.String())
	out.WriteString(" ")
	out.WriteString(rs.Body.String())

	return out.String()
}

// HashPair is a key and value in an object literal, or a spread
// (`...other`) as Key with a nil Value.
type HashPair struct {
	Key   Expression
	Value Expression
}

type HashLiteral struct {
	Token token.Token
	Pairs map[string]HashPair
	Order []string // keys of Pairs in source order
}

func (hl *HashLiteral) expressionNode()      {}
func (hl *HashLiteral) TokenLiteral() string { return hl.Token.Literal }
func (hl *HashLiteral) String() string {
	var out bytes.Buffer

	pairs := []string{}
	for _, name := range hl.Order {
		pair := hl.Pairs[name]
		if pair.Value == nil {
			pairs = append(pairs, nodeString(pair.Key))
			continue
		}
		pairs = append(pairs, nodeString(pair.Key)+": "+nodeString(pair.Value))
	}

	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")

	return out.String()
}

type DotExpression struct {
	Token    token.Token
	Left     Expression
	Property *Identifier

	// Optional is set for left?.property, which is null when left is null
	Optional bool
}

func (de *DotExpression) expressionNode()      {}
func (de *DotExpression) TokenLiteral() string { return de.Token.Literal }
func (de *DotExpression) String() string {
	var out bytes.Buffer

	out.WriteString(nodeString(de.Left))
	out.WriteString(de.Token.Literal)
	out.WriteString(de.Property.String())

	return out.String()
}

type MethodCallExpression struct {
	Token     token.Token
	Object    Expression
	Method    *Identifier
	Arguments []Expression

	// Optional is set for object?.method(), which is null when object is
	// null
	Optional bool
}

func (mce *MethodCallExpression) expressionNode()      {}
func (mce *MethodCallExpression) TokenLiteral() string { return mce.Token.Literal }
func (mce *MethodCallExpression) String() string {
	var out bytes.Buffer

	args := []string{}
	for _, a := range mce.Arguments {
		args = append(args, a.String())
	}

	out.WriteString(nodeString(mce.Object))
	out.WriteString(mce.Token.Literal)
	out.WriteString(mce.Method.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")

	return out.String()
}

type AssignmentExpression struct {
	Token token.Token
	Left  Expression
	Value Expression

	// Operator is set for compound assignments: "+" for +=, "??" for ??=
	Operator string
}

func (ae *AssignmentExpression) expressionNode()      {}
func (ae *AssignmentExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AssignmentExpression) String() string {
	var out bytes.Buffer

	// Either side is missing in code that failed to parse
	out.WriteString(nodeString(ae.Left))
	out.WriteString(" " + ae.Operator + "= ")
	out.WriteString(nodeString(ae.Value))

	return out.String()
}

type BreakStatement struct {
	Token token.Token
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) String() string {
	return bs.TokenLiteral()
}

type ContinueStatement struct {
	Token token.Token
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) String() string {
	return cs.TokenLiteral()
}

type FunctionStatement struct {
	Token      token.Token
	Name       *Identifier
	Parameters []*Identifier
	Defaults   []Expression // default value of each parameter, or nil
	Rest       bool         // the last parameter collects the extra arguments
	Patterns   []*Pattern   // a pattern taking a parameter apart, or nil
	Body       *BlockStatement
}

func (fs *FunctionStatement) statementNode()       {}
func (fs *FunctionStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *FunctionStatement) String() string {
	var out bytes.Buffer

	params := ParameterStrings(fs.Parameters, fs.Defaults, fs.Rest)

	out.WriteString(fs.TokenLiteral() + " ")
	out.WriteString(fs.Name.String())
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") ")
	out.WriteString(fs.Body.String())

	return out.String()
}

// AwaitExpression waits for a promise: `await value`.
type AwaitExpression struct {
	Token token.Token
	Value Expression
}

func (ae *AwaitExpression) expressionNode()      {}
func (ae *AwaitExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AwaitExpression) String() string {
	return "await " + nodeString(ae.Value)
}

// TryExpression runs Body and, if it fails, Catch with the error in Name:
// `try { ... } catch err { ... } finally { ... }`. Name is nil when the
// error is unused. Catch or Finally is nil when it is left out, and Finally
// runs however the try ends. Its value is the value of Body or Catch.
type TryExpression struct {
	Token   token.Token
	Body    *BlockStatement
	Name    *Identifier
	Catch   *BlockStatement
	Finally *BlockStatement
}

func (ts *TryExpression) expressionNode()      {}
func (ts *TryExpression) TokenLiteral() string { return ts.Token.Literal }
func (ts *TryExpression) String() string {
	out := "try " + ts.Body.String()
	if ts.Catch != nil {
		out += " catch "
		if ts.Name != nil {
			out += ts.Name.Value + " "
		}
		out += ts.Catch.String()
	}
	if ts.Finally != nil {
		out += " finally " + ts.Finally.String()
	}
	return out
}

// ThrowStatement raises an error: `throw "message"`.
type ThrowStatement struct {
	Token token.Token
	Value Expression
}

func (ts *ThrowStatement) statementNode()       {}
func (ts *ThrowStatement) TokenLiteral() string { return ts.Token.Literal }
func (ts *ThrowStatement) String() string {
	return "throw " + nodeString(ts.Value)
}

// ThrowExpression is throw used where a value goes, as in
// `value ?? throw "missing"` or `_ => throw "unknown"`. It never gives one.
type ThrowExpression struct {
	Token token.Token
	Value Expression
}

func (te *ThrowExpression) expressionNode()      {}
func (te *ThrowExpression) TokenLiteral() string { return te.Token.Literal }
func (te *ThrowExpression) String() string {
	return "throw " + nodeString(te.Value)
}

// Pattern takes values apart in a declaration: `[a, b]` takes the elements
// of an array in order, and `{a, b}` takes the properties of an object with
// those names.
type Pattern struct {
	Token  token.Token
	Object bool
	Names  []*Identifier

	// Rest gets what the names don't take, in `[first, ...rest]` or
	// `{name, ...others}`
	Rest *Identifier

	// Defaults has the value of each name when what it takes is null, as
	// in `{port = 8080}`, or nil; it's empty when no name has one
	Defaults []Expression

	// Keys has the property each name of an object pattern takes, when it
	// isn't the name itself, as in `{first: name}`, or ""; it's empty when
	// no name has one
	Keys []string
}

// Key returns the property the name at i takes in an object pattern.
func (p *Pattern) Key(i int) string {
	if i < len(p.Keys) && p.Keys[i] != "" {
		return p.Keys[i]
	}
	return p.Names[i].Value
}

// Default returns the default value of the name at i, or nil.
func (p *Pattern) Default(i int) Expression {
	if i < len(p.Defaults) {
		return p.Defaults[i]
	}
	return nil
}

func (p *Pattern) String() string {
	names := []string{}
	for i, n := range p.Names {
		name := n.Value
		if key := p.Key(i); key != name {
			name = strfmt.Quote(key) + ": " + name
		}
		if d := p.Default(i); d != nil {
			name += " = " + d.String()
		}
		names = append(names, name)
	}
	if p.Rest != nil {
		names = append(names, "..."+p.Rest.Value)
	}
	if p.Object {
		return "{" + strings.Join(names, ", ") + "}"
	}
	return "[" + strings.Join(names, ", ") + "]"
}

// bindingString prints what a declaration declares: a name or a pattern.
func bindingString(name *Identifier, pattern *Pattern) string {
	if pattern != nil {
		return pattern.String()
	}
	return name.String()
}

// ParameterStrings prints parameters with their default values, and the
// last one as ...name when rest is set.
func ParameterStrings(params []*Identifier, defaults []Expression, rest bool) []string {
	out := []string{}
	for i, p := range params {
		if rest && i == len(params)-1 {
			out = append(out, "..."+p.String())
			continue
		}
		if i < len(defaults) && defaults[i] != nil {
			out = append(out, p.String()+" = "+defaults[i].String())
		} else {
			out = append(out, p.String())
		}
	}
	return out
}

// SpreadExpression puts the elements of an array in an array or in the
// arguments of a call, or the properties of an object in an object:
// `...value`.
type SpreadExpression struct {
	Token token.Token
	Value Expression
}

func (se *SpreadExpression) expressionNode()      {}
func (se *SpreadExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SpreadExpression) String() string       { return "..." + nodeString(se.Value) }

// MatchExpression picks the first arm whose patterns match Subject:
//
//	match n {
//	    0 => "none"
//	    1, 2 => "a few"
//	    3..9 => "several"
//	    _ => "many"
//	}
type MatchExpression struct {
	Token   token.Token
	Subject Expression
	Arms    []*MatchArm
}

// MatchArm is one `patterns => body` of a match. A nil Patterns is `_`,
// which matches anything. Body is an expression or a block.
type MatchArm struct {
	Patterns []Expression
	Body     Node

	// Guard, in `patterns if guard => body`, must also be truthy for the
	// arm to be chosen, or nil
	Guard Expression
}

func (me *MatchExpression) expressionNode()      {}
func (me *MatchExpression) TokenLiteral() string { return me.Token.Literal }
func (me *MatchExpression) String() string {
	arms := []string{}
	for _, arm := range me.Arms {
		patterns := "_"
		if arm.Patterns != nil {
			list := []string{}
			for _, p := range arm.Patterns {
				list = append(list, p.String())
			}
			patterns = strings.Join(list, ", ")
		}
		if arm.Guard != nil {
			patterns += " if " + arm.Guard.String()
		}
		arms = append(arms, patterns+" => "+arm.Body.String())
	}
	return "match " + me.Subject.String() + " { " + strings.Join(arms, "; ") + " }"
}

// FormatExpression formats a value inside a string: ${price:.2f}.
type FormatExpression struct {
	Token token.Token
	Value Expression
	Spec  string // the format without its %, like .2f
}

func (fe *FormatExpression) expressionNode()      {}
func (fe *FormatExpression) TokenLiteral() string { return fe.Token.Literal }
func (fe *FormatExpression) String() string       { return nodeString(fe.Value) + ":" + fe.Spec }

// StructStatement declares a struct: `struct Point { x, y = 0, fn dist() { ... } }`.
type StructStatement struct {
	Token    token.Token
	Name     *Identifier
	Fields   []*Identifier
	Defaults []Expression // default value of each field, or nil
	Methods  []*FunctionStatement
}

func (ss *StructStatement) statementNode()       {}
func (ss *StructStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *StructStatement) String() string {
	parts := ParameterStrings(ss.Fields, ss.Defaults, false)
	for _, m := range ss.Methods {
		parts = append(parts, m.String())
	}
	return "struct " + ss.Name.String() + " { " + strings.Join(parts, ", ") + " }"
}

// EnumStatement declares an enum: `enum Status { Active, Blocked }`.
type EnumStatement struct {
	Token  token.Token
	Name   *Identifier
	Values []*Identifier
}

func (es *EnumStatement) statementNode()       {}
func (es *EnumStatement) TokenLiteral() string { return es.Token.Literal }
func (es *EnumStatement) String() string {
	return "enum " + es.Name.Value + " { " + strings.Join(ParameterStrings(es.Values, nil, false), ", ") + " }"
}
