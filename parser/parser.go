package parser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/token"
)

const (
	_ int = iota
	LOWEST
	PIPE        // |>
	NULLISH     // ??
	OR          // ||
	AND         // &&
	EQUALS      // ==
	LESSGREATER // > or <
	RANGE       // ..
	SUM         // +
	PRODUCT     // *
	POWER       // **
	PREFIX      // -X or !X
	CALL        // myFunction(X)
	INDEX       // array[index]
	ASSIGN      // =
)

var precedences = map[token.TokenType]int{
	token.EQ:        EQUALS,
	token.NOT_EQ:    EQUALS,
	token.LT:        LESSGREATER,
	token.GT:        LESSGREATER,
	token.LT_EQ:     LESSGREATER,
	token.IN:        LESSGREATER,
	token.GT_EQ:     LESSGREATER,
	token.RANGE:     RANGE,
	token.PLUS:      SUM,
	token.MINUS:     SUM,
	token.SLASH:     PRODUCT,
	token.ASTERISK:  PRODUCT,
	token.POWER:     POWER,
	token.MODULO:    PRODUCT,
	token.XOR:       SUM,
	token.LPAREN:    CALL,
	token.LBRACKET:  INDEX,
	token.DOT:       INDEX,
	token.ASSIGN:    ASSIGN,
	token.OP_ASSIGN: ASSIGN,
	token.AND:       AND,
	token.OR:        OR,
	token.NULLISH:   NULLISH,
	token.PIPE:      PIPE,
	token.OPTIONAL:  INDEX,

	// Postfix x++ and x-- bind to their operand like indexing
	token.INCREMENT: INDEX,
	token.DECREMENT: INDEX,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	l      *lexer.Lexer
	errors []Error

	prevToken token.Token
	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn

	// Number of errors already handled by statement recovery
	recovered int

	// Patterns of the parameters being parsed, as parseFunctionParameters
	// returns enough already
	patterns []*ast.Pattern

	// inMatchPatterns is set while parsing the patterns of a match arm,
	// where => ends the patterns instead of starting an arrow function
	inMatchPatterns bool
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []Error{},
	}

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.AWAIT, p.parseAwaitExpression)
	p.registerPrefix(token.SPREAD, p.parseSpreadExpression)
	p.registerPrefix(token.TRY, p.parseTryExpression)
	p.registerPrefix(token.MATCH, p.parseMatchExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.NULL, p.parseNullLiteral)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.TEMPLATE, p.parseTemplateLiteral)
	p.registerPrefix(token.THROW, p.parseThrowExpression)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.POWER, p.parseInfixExpression)
	p.registerInfix(token.MODULO, p.parseInfixExpression)
	p.registerInfix(token.XOR, p.parseInfixExpression)
	p.registerInfix(token.RANGE, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.IN, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.GT_EQ, p.parseInfixExpression)
	p.registerInfix(token.LT_EQ, p.parseInfixExpression)
	p.registerInfix(token.AND, p.parseInfixExpression)
	p.registerInfix(token.OR, p.parseInfixExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)
	p.registerInfix(token.DOT, p.parseDotExpression)
	p.registerInfix(token.OPTIONAL, p.parseDotExpression)
	p.registerInfix(token.NULLISH, p.parseInfixExpression)
	p.registerInfix(token.PIPE, p.parsePipeExpression)
	p.registerInfix(token.ASSIGN, p.parseAssignmentExpression)
	p.registerInfix(token.OP_ASSIGN, p.parseAssignmentExpression)
	p.registerInfix(token.INCREMENT, p.parsePostfixExpression)
	p.registerInfix(token.DECREMENT, p.parsePostfixExpression)

	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.prevToken = p.curToken
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(token.EOF) {
		stmt := p.parseStatementOrRecover(false)
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

// parseStatementOrRecover parses a statement and, if it fails, keeps only its
// first error (the later ones are usually caused by it) and skips the rest of
// the line so parsing resumes at the next statement. Returns nil on failure.
func (p *Parser) parseStatementOrRecover(inBlock bool) ast.Statement {
	start := len(p.errors)
	stmt := p.parseStatement()

	// Errors already recovered inside a nested block are kept
	if p.recovered > start {
		start = p.recovered
	}
	if len(p.errors) == start {
		return stmt
	}

	p.errors = p.errors[:start+1]
	p.skipRestOfLine(inBlock)
	p.recovered = len(p.errors)
	return nil
}

// skipRestOfLine advances to the last token on the current line, stopping
// before a closing brace when inside a block so the block can still end.
func (p *Parser) skipRestOfLine(inBlock bool) {
	if inBlock && p.curTokenIs(token.RBRACE) {
		return
	}
	line := p.curToken.Line
	for !p.peekTokenIs(token.EOF) && p.peekToken.Line == line {
		if inBlock && p.peekTokenIs(token.RBRACE) {
			return
		}
		p.nextToken()
	}
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		return p.parseLetStatement()
	case token.CONST:
		return p.parseConstStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.LOAD:
		return p.parseLoadStatement()
	case token.REPEAT:
		return p.parseRepeatStatement()
	case token.THROW:
		return p.parseThrowStatement()
	case token.BREAK:
		return p.parseBreakStatement()
	case token.CONTINUE:
		return p.parseContinueStatement()
	case token.SEMICOLON:
		// Empty statement, e.g. after a block: fn f() { ... };
		return nil
	case token.FUNCTION:
		if p.peekTokenIs(token.IDENT) {
			return p.parseFunctionStatement()
		}
		if stmt := p.parseExpressionStatement(); stmt != nil {
			return stmt
		}
		return nil
	default:
		if hint := p.otherLanguageStatement(); hint != "" {
			p.errorAt(p.curToken.Line, p.curToken.Column, "%s", hint)
			return nil
		}
		if stmt := p.parseExpressionStatement(); stmt != nil {
			return stmt
		}
		return nil
	}
}

// otherLanguageStatements are the words other languages start statements
// with, and how TargaScript says the same.
var otherLanguageStatements = map[string]string{
	"for":      "TargaScript loops with repeat: repeat item in list { ... } or repeat i in 1..10 { ... }",
	"foreach":  "TargaScript loops with repeat: repeat item in list { ... }",
	"while":    "TargaScript loops with repeat: repeat condition { ... }",
	"function": "functions are declared with fn: fn name(a, b) { ... }",
	"def":      "functions are declared with fn: fn name(a, b) { ... }",
	"func":     "functions are declared with fn: fn name(a, b) { ... }",
	"var":      "variables are declared with let, or const for ones that don't change",
}

// otherLanguageStatement returns a hint when the statement starts like one
// in another language, such as `for i in ...` or `var x = 1`: a word that
// isn't TargaScript followed, on the same line, by a name or a value.
func (p *Parser) otherLanguageStatement() string {
	if !p.curTokenIs(token.IDENT) || p.peekToken.Line != p.curToken.Line {
		return ""
	}
	hint, ok := otherLanguageStatements[p.curToken.Literal]
	if !ok {
		return ""
	}
	switch p.peekToken.Type {
	case token.IDENT, token.INT, token.FLOAT, token.STRING, token.TRUE, token.FALSE, token.BANG:
		return hint
	case token.LPAREN:
		// while (x < 3) { ... } and for (...), but not a call like f(x)
		if p.curToken.Literal == "while" || p.curToken.Literal == "for" {
			return hint
		}
	}
	return ""
}

func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.curToken}

	if p.peekTokenIs(token.ILLEGAL) && isStartingWithDigit(p.peekToken.Literal) {
		p.nextToken()
		p.errorAt(p.curToken.Line, p.curToken.Column, "invalid variable name %q: names can't start with a digit", p.curToken.Literal)
		return nil
	}

	if p.peekTokenIs(token.LBRACKET) || p.peekTokenIs(token.LBRACE) {
		p.nextToken()
		if stmt.Pattern = p.parsePattern(); stmt.Pattern == nil {
			return nil
		}
	} else {
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// literalPattern turns an array or object literal of names, like [x, y] or
// {name, age}, into a pattern, or returns nil.
func literalPattern(expression ast.Expression) *ast.Pattern {
	switch literal := expression.(type) {
	case *ast.ArrayLiteral:
		pattern := &ast.Pattern{Token: literal.Token}
		for i, element := range literal.Elements {
			if spread, ok := element.(*ast.SpreadExpression); ok && i == len(literal.Elements)-1 {
				if pattern.Rest, ok = spread.Value.(*ast.Identifier); !ok {
					return nil
				}
				continue
			}
			// [a, b = 2] reads b = 2 as an assignment
			if assign, ok := element.(*ast.AssignmentExpression); ok && assign.Operator == "" {
				if ident, ok := assign.Left.(*ast.Identifier); ok {
					pattern.Names = append(pattern.Names, ident)
					setDefault(pattern, len(pattern.Names)-1, assign.Value)
					continue
				}
			}
			ident, ok := element.(*ast.Identifier)
			if !ok {
				return nil
			}
			pattern.Names = append(pattern.Names, ident)
		}
		return pattern
	case *ast.HashLiteral:
		pattern := &ast.Pattern{Token: literal.Token, Object: true}
		for i, name := range literal.Order {
			pair := literal.Pairs[name]
			if spread, ok := pair.Key.(*ast.SpreadExpression); ok && i == len(literal.Order)-1 {
				if pattern.Rest, ok = spread.Value.(*ast.Identifier); !ok {
					return nil
				}
				continue
			}
			key, ok := pair.Key.(*ast.Identifier)
			if !ok {
				return nil
			}
			if def, ok := patternDefault(pair); ok {
				pattern.Names = append(pattern.Names, key)
				setDefault(pattern, len(pattern.Names)-1, def)
				continue
			}
			value, isIdent := pair.Value.(*ast.Identifier)
			if !isIdent || key != value {
				return nil
			}
			pattern.Names = append(pattern.Names, key)
		}
		return pattern
	}
	return nil
}

// patternDefault returns the default of a pair written {name = value} in an
// object literal, which is only valid as a pattern.
func patternDefault(pair ast.HashPair) (ast.Expression, bool) {
	assign, ok := pair.Value.(*ast.AssignmentExpression)
	if !ok || assign.Left != pair.Key {
		return nil, false
	}
	return assign.Value, true
}

// PatternOnly reports whether an object literal pair is {name = value},
// which only works as a pattern.
func PatternOnly(pair ast.HashPair) bool {
	_, ok := patternDefault(pair)
	return ok
}

// setDefault sets the default value of the name at i in a pattern.
func setDefault(pattern *ast.Pattern, i int, value ast.Expression) {
	for len(pattern.Defaults) <= i {
		pattern.Defaults = append(pattern.Defaults, nil)
	}
	pattern.Defaults[i] = value
}

// parsePattern parses `[a, b]` or `{a, b}` after let or const, with the
// opening bracket as the current token.
func (p *Parser) parsePattern() *ast.Pattern {
	pattern := &ast.Pattern{Token: p.curToken, Object: p.curTokenIs(token.LBRACE)}
	closing := token.TokenType(token.RBRACKET)
	if pattern.Object {
		closing = token.RBRACE
	}

	for !p.peekTokenIs(closing) {
		if p.peekTokenIs(token.SPREAD) {
			p.nextToken()
			if !p.expectPeek(token.IDENT) {
				return nil
			}
			pattern.Rest = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			if !p.peekTokenIs(closing) {
				p.errorAt(p.peekToken.Line, p.peekToken.Column, "...%s must be the last name in %s", pattern.Rest.Value, pattern.String())
				return nil
			}
			break
		}
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		pattern.Names = append(pattern.Names, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
		if p.peekTokenIs(token.ASSIGN) {
			p.nextToken()
			p.nextToken()
			value := p.parseExpression(LOWEST)
			if value == nil {
				return nil
			}
			setDefault(pattern, len(pattern.Names)-1, value)
		}
		if !p.peekTokenIs(closing) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}
	p.nextToken()

	if len(pattern.Names) == 0 && pattern.Rest == nil {
		p.errorAt(pattern.Token.Line, pattern.Token.Column, "expected at least one name in %s", pattern.String())
		return nil
	}
	return pattern
}

func (p *Parser) parseConstStatement() *ast.ConstStatement {
	stmt := &ast.ConstStatement{Token: p.curToken}

	if p.peekTokenIs(token.ILLEGAL) && isStartingWithDigit(p.peekToken.Literal) {
		p.nextToken()
		p.errorAt(p.curToken.Line, p.curToken.Column, "invalid constant name %q: names can't start with a digit", p.curToken.Literal)
		return nil
	}

	if p.peekTokenIs(token.LBRACKET) || p.peekTokenIs(token.LBRACE) {
		p.nextToken()
		if stmt.Pattern = p.parsePattern(); stmt.Pattern == nil {
			return nil
		}
	} else {
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func isStartingWithDigit(s string) bool {
	if len(s) == 0 {
		return false
	}
	firstChar := []rune(s)[0]
	return '0' <= firstChar && firstChar <= '9'
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	// A bare return ends at "}", ";", the end of input or the end of the line
	if p.peekTokenIs(token.RBRACE) || p.peekTokenIs(token.SEMICOLON) ||
		p.peekTokenIs(token.EOF) || p.peekToken.Line > p.curToken.Line {
		if p.peekTokenIs(token.SEMICOLON) {
			p.nextToken()
		}
		return stmt
	}

	p.nextToken()

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)

	// print "hi": a name followed by a value on the same line is almost
	// always a call without parentheses
	if ident, ok := stmt.Expression.(*ast.Identifier); ok && p.peekToken.Line == p.curToken.Line {
		switch p.peekToken.Type {
		case token.STRING, token.TEMPLATE, token.INT, token.FLOAT:
			p.errorAt(p.peekToken.Line, p.peekToken.Column, "calls need parentheses: %s(...)", ident.Value)
			return nil
		}
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) &&
		!isStatementEnd(p.peekToken.Type) &&
		precedence < p.peekPrecedence() {

		if p.peekStartsNewLine() {
			return leftExp
		}

		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

// peekStartsNewLine reports whether the next token begins a new statement
// rather than continuing the current expression. A "(", "[" or "-" at the
// start of a line could do either, so it is treated as a new statement:
//
//	let a = b
//	(c + d).print()
//
// To continue an expression on the next line, end the line with an operator.
func (p *Parser) peekStartsNewLine() bool {
	if p.peekToken.Line <= p.curToken.Line {
		return false
	}
	switch p.peekToken.Type {
	case token.LPAREN, token.LBRACKET, token.MINUS:
		return true
	default:
		return false
	}
}

func isStatementEnd(tokenType token.TokenType) bool {
	switch tokenType {
	case token.RBRACE, token.EOF, token.LET, token.RETURN, token.IF, token.FUNCTION, token.CONST, token.REPEAT, token.SEMICOLON:
		return true
	default:
		return false
	}
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}

	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}

	return LOWEST
}

func (p *Parser) parseIdentifier() ast.Expression {
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	// x => x * 2
	if p.peekTokenIs(token.ARROW) && !p.inMatchPatterns {
		return p.parseArrowFunction(ident.Token, []*ast.Identifier{ident}, []ast.Expression{nil}, nil)
	}
	return ident
}

// parseArrowFunction parses the body of an arrow function, after its
// parameters, with => as the next token: an expression, or a block.
func (p *Parser) parseArrowFunction(tok token.Token, params []*ast.Identifier, defaults []ast.Expression, patterns []*ast.Pattern) ast.Expression {
	p.nextToken() // =>
	lit := &ast.FunctionLiteral{Token: tok, Parameters: params, Defaults: defaults, Patterns: patterns, Arrow: true}
	p.nextToken()

	if p.curTokenIs(token.LBRACE) && !p.startsObject() {
		lit.Body = p.parseBlockStatement()
		return lit
	}

	// Inside an arrow function's body, => belongs to another arrow function
	outer := p.inMatchPatterns
	p.inMatchPatterns = false
	body := p.parseExpression(LOWEST)
	p.inMatchPatterns = outer
	if body == nil {
		return nil
	}
	lit.Body = &ast.BlockStatement{Token: tok, Statements: []ast.Statement{
		&ast.ExpressionStatement{Token: tok, Expression: body},
	}}
	return lit
}

// startsObject reports whether the { at the current token starts an object
// like {name: 1}, {x, y} or {...other} rather than a block. No block starts
// with a spread, or with a name or a string followed by a colon or a comma.
// {x} stays a block that returns x.
func (p *Parser) startsObject() bool {
	if p.peekTokenIs(token.SPREAD) {
		return true
	}
	if !p.peekTokenIs(token.IDENT) && !p.peekTokenIs(token.STRING) {
		return false
	}
	saved := *p.l
	next := p.l.NextToken()
	*p.l = saved
	return next.Type == token.COLON || next.Type == token.COMMA
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}

	text := p.curToken.Literal
	base := 10
	if len(text) > 1 && text[0] == '0' && strings.ContainsRune("xXbBoO", rune(text[1])) {
		// Base 0 reads the prefix; plain numbers stay decimal, so 010 is 10
		base = 0
	} else {
		text = strings.ReplaceAll(text, "_", "")
	}
	value, err := strconv.ParseInt(text, base, 64)
	if errors.Is(err, strconv.ErrRange) {
		p.errorAt(p.curToken.Line, p.curToken.Column, "%s is too large for an integer, the limit is 9223372036854775807", p.curToken.Literal)
		return nil
	}
	if err != nil {
		p.errorAt(p.curToken.Line, p.curToken.Column, "%s is not a valid number", p.curToken.Literal)
		return nil
	}

	lit.Value = value

	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}

	value, err := strconv.ParseFloat(strings.ReplaceAll(p.curToken.Literal, "_", ""), 64)
	if err != nil {
		p.errorAt(p.curToken.Line, p.curToken.Column, "%s is too large for a number", p.curToken.Literal)
		return nil
	}

	lit.Value = value

	return lit
}

func (p *Parser) parseSpreadExpression() ast.Expression {
	expression := &ast.SpreadExpression{Token: p.curToken}
	p.nextToken()
	expression.Value = p.parseExpression(PREFIX)
	return expression
}

func (p *Parser) parseAwaitExpression() ast.Expression {
	expression := &ast.AwaitExpression{Token: p.curToken}
	p.nextToken()
	expression.Value = p.parseExpression(PREFIX)
	return expression
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	// === and !== from JavaScript
	if (p.curTokenIs(token.EQ) || p.curTokenIs(token.NOT_EQ)) && p.peekTokenIs(token.ASSIGN) &&
		p.peekToken.Line == p.curToken.Line && p.peekToken.Column == p.curToken.Column+2 {
		p.errorAt(p.curToken.Line, p.curToken.Column, "TargaScript compares with %s, which already checks the type: 1 == \"1\" is false", p.curToken.Literal)
		return nil
	}

	precedence := p.curPrecedence()
	if p.curTokenIs(token.POWER) {
		// ** groups to the right: 2 ** 3 ** 2 is 2 ** (3 ** 2)
		precedence--
	}
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseNullLiteral() ast.Expression {
	return &ast.NullLiteral{Token: p.curToken}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	open := p.curToken

	// () => value
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		if !p.peekTokenIs(token.ARROW) {
			p.errorAt(open.Line, open.Column, "expected an expression in ()")
			return nil
		}
		return p.parseArrowFunction(open, []*ast.Identifier{}, []ast.Expression{}, nil)
	}

	p.nextToken()
	exps := []ast.Expression{p.parseExpression(LOWEST)}
	// (a, b) can only be the parameters of an arrow function
	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		exps = append(exps, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if p.peekTokenIs(token.ARROW) && !p.inMatchPatterns {
		params := []*ast.Identifier{}
		defaults := []ast.Expression{}
		patterns := []*ast.Pattern{}
		for _, exp := range exps {
			// ([x, y]) => ... and ({name}) => ... take the argument apart
			if pattern := literalPattern(exp); pattern != nil {
				params = append(params, &ast.Identifier{Token: pattern.Token, Value: pattern.String()})
				defaults = append(defaults, nil)
				patterns = append(patterns, pattern)
				continue
			}
			patterns = append(patterns, nil)
			switch exp := exp.(type) {
			case *ast.Identifier:
				params = append(params, exp)
				defaults = append(defaults, nil)
			case *ast.AssignmentExpression:
				// (a, b = 1) => ...
				if ident, ok := exp.Left.(*ast.Identifier); ok && exp.Operator == "" {
					params = append(params, ident)
					defaults = append(defaults, exp.Value)
					continue
				}
				p.errorAt(open.Line, open.Column, "expected parameter names before =>")
				return nil
			default:
				p.errorAt(open.Line, open.Column, "expected parameter names before =>")
				return nil
			}
		}
		return p.parseArrowFunction(open, params, defaults, patterns)
	}

	if len(exps) > 1 {
		p.errorAt(open.Line, open.Column, "expected one expression in (), found %d separated by commas", len(exps))
		return nil
	}
	return exps[0]
}

func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{Token: p.curToken}

	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.Consequence = p.parseBlockStatement()

	if p.peekTokenIs(token.ELSE) {
		p.nextToken()

		if p.peekTokenIs(token.IF) {
			p.nextToken()

			nestedIf := p.parseIfExpression()
			if nestedIf == nil {
				return nil
			}

			alternative := &ast.BlockStatement{
				Token: p.curToken,
				Statements: []ast.Statement{
					&ast.ExpressionStatement{
						Token:      p.curToken,
						Expression: nestedIf,
					},
				},
			}

			expression.Alternative = alternative
		} else if p.expectPeek(token.LBRACE) {
			expression.Alternative = p.parseBlockStatement()
		} else {
			return nil
		}
	}

	return expression
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatementOrRecover(true)
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	if p.curTokenIs(token.EOF) {
		p.errorAt(block.Token.Line, block.Token.Column, "this \"{\" is never closed, add a \"}\" to end the block")
	}

	return block
}

func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curToken}

	if p.peekTokenIs(token.IDENT) {
		p.nextToken()
		lit.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	outer := p.patterns
	p.patterns = nil
	lit.Parameters, lit.Defaults, lit.Rest = p.parseFunctionParameters()
	lit.Patterns = p.patterns
	p.patterns = outer

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()

	return lit
}

// parseFunctionParameters parses the parameters of a function, their
// default values, nil for those without one, and whether the last one is a
// rest parameter (...name).
func (p *Parser) parseFunctionParameters() ([]*ast.Identifier, []ast.Expression, bool) {
	identifiers := []*ast.Identifier{}
	defaults := []ast.Expression{}
	rest := false

	for !p.peekTokenIs(token.RPAREN) {
		p.nextToken()

		// A pattern takes the argument apart: fn area({width, height})
		if p.curTokenIs(token.LBRACKET) || p.curTokenIs(token.LBRACE) {
			pattern := p.parsePattern()
			if pattern == nil {
				return nil, nil, false
			}
			p.patterns = append(p.patterns, pattern)
			identifiers = append(identifiers, &ast.Identifier{Token: pattern.Token, Value: pattern.String()})
			var defaultValue ast.Expression
			if p.peekTokenIs(token.ASSIGN) {
				p.nextToken()
				p.nextToken()
				if defaultValue = p.parseExpression(LOWEST); defaultValue == nil {
					return nil, nil, false
				}
			}
			defaults = append(defaults, defaultValue)
			if !p.peekTokenIs(token.COMMA) {
				break
			}
			p.nextToken()
			continue
		}
		p.patterns = append(p.patterns, nil)

		if p.curTokenIs(token.SPREAD) {
			rest = true
			p.nextToken()
		}

		if p.curTokenIs(token.ILLEGAL) && isStartingWithDigit(p.curToken.Literal) {
			p.errorAt(p.curToken.Line, p.curToken.Column, "invalid parameter name %q: names can't start with a digit", p.curToken.Literal)
			return nil, nil, false
		}
		if !p.curTokenIs(token.IDENT) {
			p.errorAt(p.curToken.Line, p.curToken.Column, "expected a parameter name, found %s", describe(p.curToken))
			return nil, nil, false
		}

		identifiers = append(identifiers, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})

		if rest {
			defaults = append(defaults, nil)
			if !p.peekTokenIs(token.RPAREN) {
				p.errorAt(p.peekToken.Line, p.peekToken.Column, "the ...%s parameter must be the last one", p.curToken.Literal)
				return nil, nil, false
			}
			break
		}

		var defaultValue ast.Expression
		if p.peekTokenIs(token.ASSIGN) {
			p.nextToken()
			p.nextToken()
			// The value ends at the comma or parenthesis after it
			if defaultValue = p.parseExpression(LOWEST); defaultValue == nil {
				return nil, nil, false
			}
		}
		defaults = append(defaults, defaultValue)

		if !p.peekTokenIs(token.COMMA) {
			break
		}
		p.nextToken()
	}

	if !p.expectPeek(token.RPAREN) {
		return nil, nil, false
	}

	return identifiers, defaults, rest
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseCallArguments()
	return exp
}

// parsePipeExpression reads value |> f as f(value), and value |> f(a) or
// value |> obj.f(a) with value as the first argument, so calls chain left
// to right.
func (p *Parser) parsePipeExpression(value ast.Expression) ast.Expression {
	pipe := p.curToken
	p.nextToken()
	target := p.parseExpression(PIPE)
	if target == nil {
		return nil
	}

	switch call := target.(type) {
	case *ast.CallExpression:
		piped := *call
		piped.Arguments = append([]ast.Expression{value}, call.Arguments...)
		return &piped
	case *ast.MethodCallExpression:
		piped := *call
		piped.Arguments = append([]ast.Expression{value}, call.Arguments...)
		return &piped
	}
	return &ast.CallExpression{Token: pipe, Function: target, Arguments: []ast.Expression{value}}
}

func (p *Parser) parseCallArguments() []ast.Expression {
	return p.parseExpressionList(token.RPAREN)
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseLoadStatement() *ast.LoadStatement {
	stmt := &ast.LoadStatement{Token: p.curToken}

	p.nextToken()

	if !p.curTokenIs(token.STRING) {
		p.errorAt(p.curToken.Line, p.curToken.Column, "expected a file name in quotes after \"load\", found %s", describe(p.curToken))
		return nil
	}

	stmt.Path = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

	// Leave the last token of the statement as the current one, like the
	// other statements, so the next statement isn't skipped
	if p.peekTokenIs(token.AS) {
		p.nextToken()
		p.nextToken()

		if !p.curTokenIs(token.IDENT) {
			p.errorAt(p.curToken.Line, p.curToken.Column, "expected a name after \"as\", found %s", describe(p.curToken))
			return nil
		}

		stmt.Alias = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseTryExpression() ast.Expression {
	stmt := &ast.TryExpression{Token: p.curToken}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()

	if !p.expectPeek(token.CATCH) {
		return nil
	}
	if p.peekTokenIs(token.IDENT) {
		p.nextToken()
		stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Catch = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseMatchExpression() ast.Expression {
	expression := &ast.MatchExpression{Token: p.curToken}

	p.nextToken()
	expression.Subject = p.parseExpression(LOWEST)
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	for !p.peekTokenIs(token.RBRACE) {
		if p.peekTokenIs(token.EOF) {
			p.errorAt(expression.Token.Line, expression.Token.Column, "this match is never closed, add a \"}\" after its last arm")
			return nil
		}
		p.nextToken()
		arm := &ast.MatchArm{}

		// _ matches anything, and leaves Patterns nil
		if !(p.curTokenIs(token.IDENT) && p.curToken.Literal == "_" && (p.peekTokenIs(token.ARROW) || p.peekTokenIs(token.IF))) {
			p.inMatchPatterns = true
			for {
				pattern := p.parseExpression(LOWEST)
				if pattern == nil {
					p.inMatchPatterns = false
					return nil
				}
				arm.Patterns = append(arm.Patterns, pattern)
				if !p.peekTokenIs(token.COMMA) {
					break
				}
				p.nextToken()
				p.nextToken()
			}
			p.inMatchPatterns = false
		}

		// patterns if guard => body
		if p.peekTokenIs(token.IF) {
			p.nextToken()
			p.nextToken()
			p.inMatchPatterns = true
			arm.Guard = p.parseExpression(LOWEST)
			p.inMatchPatterns = false
			if arm.Guard == nil {
				return nil
			}
		}

		if !p.expectPeek(token.ARROW) {
			return nil
		}
		p.nextToken()
		// Braces start a block, unless they hold an object: => {a: 1}
		if p.curTokenIs(token.LBRACE) && !p.startsObject() {
			arm.Body = p.parseBlockStatement()
		} else {
			arm.Body = p.parseExpression(LOWEST)
		}
		if arm.Body == nil {
			return nil
		}
		expression.Arms = append(expression.Arms, arm)

		// Arms can be separated by commas as well as by new lines
		if p.peekTokenIs(token.COMMA) {
			p.nextToken()
		}
	}
	p.nextToken()

	return expression
}

// parseThrowExpression parses throw where a value goes, which raises the
// error instead of giving one.
func (p *Parser) parseThrowExpression() ast.Expression {
	expression := &ast.ThrowExpression{Token: p.curToken}
	p.nextToken()
	if expression.Value = p.parseExpression(LOWEST); expression.Value == nil {
		return nil
	}
	return expression
}

func (p *Parser) parseThrowStatement() ast.Statement {
	stmt := &ast.ThrowStatement{Token: p.curToken}
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)
	return stmt
}

func (p *Parser) parseRepeatStatement() *ast.RepeatStatement {
	stmt := &ast.RepeatStatement{Token: p.curToken}

	p.nextToken()

	if p.curTokenIs(token.IDENT) && p.peekTokenIs(token.COMMA) {
		stmt.Key = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		p.nextToken()
		if p.peekTokenIs(token.LBRACKET) || p.peekTokenIs(token.LBRACE) {
			p.nextToken()
			if stmt.Pattern = p.parsePattern(); stmt.Pattern == nil {
				return nil
			}
			stmt.Iterator = &ast.Identifier{Token: stmt.Pattern.Token, Value: stmt.Pattern.String()}
		} else {
			if !p.expectPeek(token.IDENT) {
				return nil
			}
			stmt.Iterator = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		}
		if !p.expectPeek(token.IN) {
			return nil
		}
		p.nextToken()
		stmt.Collection = p.parseExpression(LOWEST)
	} else if p.curTokenIs(token.IDENT) && p.peekTokenIs(token.IN) {
		stmt.Iterator = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		p.nextToken()
		p.nextToken()
		stmt.Collection = p.parseExpression(LOWEST)
	} else {
		stmt.Iterator = nil
		stmt.Collection = p.parseExpression(LOWEST)

		// A literal before in is a pattern: repeat [x, y] in points
		if in, ok := stmt.Collection.(*ast.InfixExpression); ok && in.Operator == "in" {
			pattern := literalPattern(in.Left)
			switch {
			case pattern != nil:
				stmt.Pattern = pattern
				stmt.Iterator = &ast.Identifier{Token: pattern.Token, Value: pattern.String()}
				stmt.Collection = in.Right
			case isLiteral(in.Left):
				p.errorAt(in.Token.Line, in.Token.Column, "expected a name or a pattern like [a, b] before \"in\"")
				return nil
			}
		}
	}

	if !p.peekTokenIs(token.LBRACE) {
		p.errorAt(p.peekToken.Line, p.peekToken.Column, "expected \"{\" to start the repeat body, found %s", describe(p.peekToken))
		return nil
	}

	p.nextToken()
	stmt.Body = p.parseBlockStatement()

	return stmt
}

// isLiteral reports whether node is an array or object literal.
func isLiteral(node ast.Expression) bool {
	switch node.(type) {
	case *ast.ArrayLiteral, *ast.HashLiteral:
		return true
	}
	return false
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}

	array.Elements = p.parseExpressionList(token.RBRACKET)

	return array
}

func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}

	// A trailing comma before the closing token is allowed
	for !p.peekTokenIs(end) {
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))

		if !p.peekTokenIs(token.COMMA) {
			break
		}
		p.nextToken()
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{
		Token: p.curToken,
		Left:  left,
	}

	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return exp
}

func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken}
	hash.Pairs = make(map[string]ast.HashPair)

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()

		key := p.parseExpression(LOWEST)

		// {name, age} is short for {"name": name, "age": age}
		if ident, ok := key.(*ast.Identifier); ok && (p.peekTokenIs(token.COMMA) || p.peekTokenIs(token.RBRACE)) {
			name := hashKeyName(key, len(hash.Order))
			if _, exists := hash.Pairs[name]; !exists {
				hash.Order = append(hash.Order, name)
			}
			hash.Pairs[name] = ast.HashPair{Key: ident, Value: ident}
			if !p.peekTokenIs(token.COMMA) {
				break
			}
			p.nextToken()
			continue
		}

		// {port = 80} is only a pattern, for parameters like
		// ({port = 80}) => ...; the compiler refuses it as an object
		if assign, ok := key.(*ast.AssignmentExpression); ok && assign.Operator == "" && (p.peekTokenIs(token.COMMA) || p.peekTokenIs(token.RBRACE)) {
			if ident, ok := assign.Left.(*ast.Identifier); ok {
				name := hashKeyName(ident, len(hash.Order))
				if _, exists := hash.Pairs[name]; !exists {
					hash.Order = append(hash.Order, name)
				}
				hash.Pairs[name] = ast.HashPair{Key: ident, Value: assign}
				if !p.peekTokenIs(token.COMMA) {
					break
				}
				p.nextToken()
				continue
			}
		}

		if spread, ok := key.(*ast.SpreadExpression); ok {
			name := fmt.Sprintf("\x00spread%d", len(hash.Order))
			hash.Order = append(hash.Order, name)
			hash.Pairs[name] = ast.HashPair{Key: spread}
			if !p.peekTokenIs(token.COMMA) {
				break
			}
			p.nextToken()
			continue
		}
		if !p.expectPeek(token.COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)

		name := hashKeyName(key, len(hash.Order))
		if _, exists := hash.Pairs[name]; !exists {
			hash.Order = append(hash.Order, name)
		}
		hash.Pairs[name] = ast.HashPair{Key: key, Value: value}

		if !p.peekTokenIs(token.COMMA) {
			break
		}
		p.nextToken()
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return hash
}

// hashKeyName returns the name a hash literal pair is stored under. Bare
// names and string literals use their text, so {a: 1, "a": 2} has one key.
// Other keys may only be equal at runtime, so each gets its own name and
// duplicates are resolved when the hash is evaluated.
func hashKeyName(key ast.Expression, index int) string {
	switch key := key.(type) {
	case *ast.Identifier:
		return key.Value
	case *ast.StringLiteral:
		return key.Value
	default:
		return fmt.Sprintf("\x00%d", index)
	}
}

func (p *Parser) parseDotExpression(left ast.Expression) ast.Expression {
	expression := &ast.DotExpression{
		Token:    p.curToken,
		Left:     left,
		Optional: p.curTokenIs(token.OPTIONAL),
	}

	p.nextToken()

	// Keywords are fine as names here, like `text.repeat(3)`
	if !p.curTokenIs(token.IDENT) && !token.IsKeyword(p.curToken) {
		p.errorAt(p.curToken.Line, p.curToken.Column, "expected a property or method name after %q, found %s", expression.Token.Literal, describe(p.curToken))
		return nil
	}

	propertyName := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if p.peekTokenIs(token.LPAREN) {
		methodCall := &ast.MethodCallExpression{
			Token:    expression.Token,
			Object:   left,
			Method:   propertyName,
			Optional: expression.Optional,
		}

		p.nextToken()

		methodCall.Arguments = p.parseExpressionList(token.RPAREN)
		if methodCall.Arguments == nil {
			return nil
		}

		return methodCall
	}

	expression.Property = propertyName
	return expression
}

func (p *Parser) parsePostfixExpression(left ast.Expression) ast.Expression {
	return &ast.PostfixExpression{Token: p.curToken, Left: left, Operator: p.curToken.Literal}
}

func (p *Parser) parseAssignmentExpression(left ast.Expression) ast.Expression {
	expr := &ast.AssignmentExpression{
		Token: p.curToken,
		Left:  left,
	}
	if p.curTokenIs(token.OP_ASSIGN) {
		expr.Operator = strings.TrimSuffix(p.curToken.Literal, "=")
	}

	p.nextToken()
	expr.Value = p.parseExpression(LOWEST)

	return expr
}

func (p *Parser) parseBreakStatement() *ast.BreakStatement {
	stmt := &ast.BreakStatement{Token: p.curToken}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseContinueStatement() *ast.ContinueStatement {
	stmt := &ast.ContinueStatement{Token: p.curToken}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseFunctionStatement() *ast.FunctionStatement {
	stmt := &ast.FunctionStatement{Token: p.curToken}

	if !p.peekTokenIs(token.IDENT) {
		p.errorAt(p.peekToken.Line, p.peekToken.Column, "expected a function name after \"fn\", found %s", describe(p.peekToken))
		return nil
	}
	p.nextToken()

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	outer := p.patterns
	p.patterns = nil
	stmt.Parameters, stmt.Defaults, stmt.Rest = p.parseFunctionParameters()
	stmt.Patterns = p.patterns
	p.patterns = outer

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

// parseTemplateLiteral splits the raw source of an interpolated string into
// text and ${...} expressions, parsing each expression on its own.
func (p *Parser) parseTemplateLiteral() ast.Expression {
	tmpl := &ast.TemplateLiteral{Token: p.curToken}
	raw := []rune(p.curToken.Literal)
	var text strings.Builder

	flushText := func() {
		if text.Len() > 0 {
			tmpl.Parts = append(tmpl.Parts, &ast.StringLiteral{Token: p.curToken, Value: text.String()})
			text.Reset()
		}
	}

	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		switch {
		case ch == '\\' && i+1 < len(raw):
			i++
			text.WriteString(unescape(raw[i]))
		case ch == '$' && i+1 < len(raw) && raw[i+1] == '{':
			end := matchingBrace(raw, i+1)
			source := string(raw[i+2 : end])
			expr := p.parseInterpolation(source)
			if expr == nil {
				return nil
			}
			flushText()
			tmpl.Parts = append(tmpl.Parts, expr)
			i = end
		default:
			text.WriteRune(ch)
		}
	}
	flushText()

	return tmpl
}

func (p *Parser) parseInterpolation(source string) ast.Expression {
	if strings.TrimSpace(source) == "" {
		p.errorAt(p.curToken.Line, p.curToken.Column, "empty ${} in string")
		return nil
	}

	// ${value:spec} formats the value like format("%spec", value)
	if expression, spec, ok := splitFormatSpec(source); ok {
		if !validFormatSpec(spec) {
			p.errorAt(p.curToken.Line, p.curToken.Column, "in ${%s}: %q is not a format like .2f, 5d or -10s", source, spec)
			return nil
		}
		value := p.parseInterpolation(expression)
		if value == nil {
			return nil
		}
		return &ast.FormatExpression{Token: p.curToken, Value: value, Spec: spec}
	}

	sub := New(lexer.New(source))
	expr := sub.parseExpression(LOWEST)
	if len(sub.errors) == 0 && !sub.peekTokenIs(token.EOF) {
		sub.errorAt(0, 0, "unexpected %s", describe(sub.peekToken))
	}
	// Positions inside the interpolation are meaningless to the user, so
	// errors point at the string
	for _, err := range sub.errors {
		p.errorAt(p.curToken.Line, p.curToken.Column, "in ${%s}: %s", source, err.Message)
	}
	if len(sub.errors) > 0 {
		return nil
	}
	return expr
}

// splitFormatSpec splits "value:spec" at its first colon outside brackets,
// parentheses, braces and strings.
func splitFormatSpec(source string) (string, string, bool) {
	depth := 0
	inString := false
	for i := 0; i < len(source); i++ {
		switch c := source[i]; {
		case inString && c == '\\':
			i++
		case c == '"':
			inString = !inString
		case inString:
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			depth--
		case c == ':' && depth == 0:
			return source[:i], source[i+1:], true
		}
	}
	return "", "", false
}

// validFormatSpec reports whether spec is a printf format without the %,
// like .2f, 5d, -10s or 08.3f.
func validFormatSpec(spec string) bool {
	if spec == "" {
		return false
	}
	for i, c := range spec {
		last := i == len(spec)-1
		switch {
		case last && strings.ContainsRune("dfFeEgGsqvxXobct", c):
			return true
		case !last && strings.ContainsRune("+-# 0123456789.", c):
		default:
			return false
		}
	}
	return false
}

// matchingBrace returns the index of the "}" closing the "{" at open,
// skipping nested braces and strings. The lexer has already checked that
// it exists.
func matchingBrace(raw []rune, open int) int {
	depth := 0
	for i := open; i < len(raw); i++ {
		switch raw[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		case '"':
			for i++; i < len(raw) && raw[i] != '"'; i++ {
				if raw[i] == '\\' {
					i++
				}
			}
		}
	}
	return len(raw) - 1
}

// unescape returns what an escape like \n stands for. Unknown escapes keep
// their backslash, as in plain strings.
func unescape(ch rune) string {
	switch ch {
	case 'n':
		return "\n"
	case 't':
		return "\t"
	case 'r':
		return "\r"
	case '"', '\\', '$':
		return string(ch)
	default:
		return "\\" + string(ch)
	}
}
