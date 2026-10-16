package parser

import (
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
	token.EQ:       EQUALS,
	token.NOT_EQ:   EQUALS,
	token.LT:       LESSGREATER,
	token.GT:       LESSGREATER,
	token.LT_EQ:    LESSGREATER,
	token.GT_EQ:    LESSGREATER,
	token.RANGE:    RANGE,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.SLASH:    PRODUCT,
	token.ASTERISK: PRODUCT,
	token.POWER:    POWER,
	token.MODULO:   PRODUCT,
	token.XOR:      SUM,
	token.LPAREN:   CALL,
	token.LBRACKET: INDEX,
	token.DOT:      INDEX,
	token.ASSIGN:   ASSIGN,
	token.AND:      AND,
	token.OR:       OR,
	token.NULLISH:  NULLISH,
	token.OPTIONAL: INDEX,

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
	p.registerInfix(token.ASSIGN, p.parseAssignmentExpression)
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
	case token.PRINT:
		return p.parsePrintStatement()
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
		return p.parseExpressionStatement()
	default:
		return p.parseExpressionStatement()
	}
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
			value, isIdent := pair.Value.(*ast.Identifier)
			if !ok || !isIdent || key != value {
				return nil
			}
			pattern.Names = append(pattern.Names, key)
		}
		return pattern
	}
	return nil
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
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
	if err != nil {
		p.errorAt(p.curToken.Line, p.curToken.Column, "%s is too large for an integer, the limit is 9223372036854775807", p.curToken.Literal)
		return nil
	}

	lit.Value = value

	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}

	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
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
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
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
		if !(p.curTokenIs(token.IDENT) && p.curToken.Literal == "_" && p.peekTokenIs(token.ARROW)) {
			for {
				pattern := p.parseExpression(LOWEST)
				if pattern == nil {
					return nil
				}
				arm.Patterns = append(arm.Patterns, pattern)
				if !p.peekTokenIs(token.COMMA) {
					break
				}
				p.nextToken()
				p.nextToken()
			}
		}

		if !p.expectPeek(token.ARROW) {
			return nil
		}
		p.nextToken()
		// Braces start a block; an object needs parentheses: => ({"a": 1})
		if p.curTokenIs(token.LBRACE) {
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

		// A literal followed by in is a pattern: repeat [x, y] in points
		if p.peekTokenIs(token.IN) {
			pattern := literalPattern(stmt.Collection)
			if pattern == nil {
				p.errorAt(p.peekToken.Line, p.peekToken.Column, "expected a name or a pattern like [a, b] before \"in\"")
				return nil
			}
			stmt.Pattern = pattern
			stmt.Iterator = &ast.Identifier{Token: pattern.Token, Value: pattern.String()}
			p.nextToken()
			p.nextToken()
			stmt.Collection = p.parseExpression(LOWEST)
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

	p.nextToken()
	expr.Value = p.parseExpression(LOWEST)

	return expr
}

func (p *Parser) parsePrintStatement() *ast.PrintStatement {
	stmt := &ast.PrintStatement{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	stmt.Arguments = p.parseExpressionList(token.RPAREN)
	if stmt.Arguments == nil {
		return nil
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
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
			text.WriteRune(unescape(raw[i]))
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

func unescape(ch rune) rune {
	switch ch {
	case 'n':
		return '\n'
	case 't':
		return '\t'
	case 'r':
		return '\r'
	default:
		return ch
	}
}
