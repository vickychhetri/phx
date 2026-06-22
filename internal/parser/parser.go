package parser

import (
	"fmt"
	"phx/internal/ast"
	"phx/internal/lexer"
	"phx/internal/token"
	"strconv"
	"strings"
)

const (
	_ int = iota
	LOWEST
	ASSIGN      // =
	TERNARY     // ? :
	EQUALS      // == or === or != or !==
	LESSGREATER // > or < or >= or <=
	SUM         // + or - or .
	PRODUCT     // * or /
	PREFIX      // -X or !X
	POSTFIX     // X++ or X--
	CALL        // myFunction(X)
)

var precedences = map[token.TokenType]int{
	token.ASSIGN:          ASSIGN,
	token.ADD_ASSIGN:      ASSIGN,
	token.SUB_ASSIGN:      ASSIGN,
	token.QUESTION:        TERNARY,
	token.EQUAL:           EQUALS,
	token.IDENTICAL:       EQUALS,
	token.NOT_EQUAL:       EQUALS,
	token.NOT_IDENTICAL:   EQUALS,
	token.LT:              LESSGREATER,
	token.GT:              LESSGREATER,
	token.LTE:             LESSGREATER,
	token.GTE:             LESSGREATER,
	token.PLUS:            SUM,
	token.MINUS:           SUM,
	token.CONCAT:          SUM,
	token.MUL:             PRODUCT,
	token.DIV:             PRODUCT,
	token.MOD:             PRODUCT,
	token.INC:             POSTFIX,
	token.DEC:             POSTFIX,
	token.LPAREN:          CALL,
	token.OBJECT_OPERATOR: CALL,
	token.LBRACKET:        CALL,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.T_VARIABLE, p.parseVariable)
	p.registerPrefix(token.T_IDENTIFIER, p.parseIdentifier)
	p.registerPrefix(token.T_LNUMBER, p.parseIntegerLiteral)
	p.registerPrefix(token.T_DNUMBER, p.parseFloatLiteral)
	p.registerPrefix(token.T_CONSTANT_ENCAPSED_STRING, p.parseStringLiteral)
	p.registerPrefix(token.T_DOUBLE_QUOTED_STRING, p.parseStringLiteral)
	p.registerPrefix(token.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(token.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(token.NULL, p.parseNullLiteral)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.NEW, p.parseNewExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.INC, p.parsePrefixExpression)
	p.registerPrefix(token.DEC, p.parsePrefixExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionExpression)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.MUL, p.parseInfixExpression)
	p.registerInfix(token.DIV, p.parseInfixExpression)
	p.registerInfix(token.MOD, p.parseInfixExpression)
	p.registerInfix(token.CONCAT, p.parseInfixExpression)
	p.registerInfix(token.EQUAL, p.parseInfixExpression)
	p.registerInfix(token.IDENTICAL, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQUAL, p.parseInfixExpression)
	p.registerInfix(token.NOT_IDENTICAL, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.LTE, p.parseInfixExpression)
	p.registerInfix(token.GTE, p.parseInfixExpression)
	p.registerInfix(token.ASSIGN, p.parseAssignExpression)
	p.registerInfix(token.ADD_ASSIGN, p.parseAssignExpression)
	p.registerInfix(token.SUB_ASSIGN, p.parseAssignExpression)
	p.registerInfix(token.QUESTION, p.parseTernaryExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.OBJECT_OPERATOR, p.parseObjectAccessExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)
	p.registerInfix(token.INC, p.parsePostExpression)
	p.registerInfix(token.DEC, p.parsePostExpression)

	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead at line %d, col %d",
		t, p.peekToken.Type, p.peekToken.Pos.Line, p.peekToken.Pos.Column)
	p.errors = append(p.errors, msg)
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for p.curToken.Type != token.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.T_INLINE_HTML:
		return p.parseInlineHTMLStatement()
	case token.T_OPEN_TAG, token.T_CLOSE_TAG:
		return nil
	case token.ECHO:
		return p.parseEchoStatement()
	case token.IF:
		return p.parseIfStatement()
	case token.LBRACE:
		return p.parseBlockStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.CLASS:
		return p.parseClassStatement()
	case token.SWITCH:
		return p.parseSwitchStatement()
	case token.FUNCTION:
		return p.parseFunctionStatement()
	case token.WHILE:
		return p.parseWhileStatement()
	case token.DO:
		return p.parseDoWhileStatement()
	case token.FOR:
		return p.parseForStatement()
	case token.BREAK:
		return p.parseBreakStatement()
	case token.CONTINUE:
		return p.parseContinueStatement()
	case token.INCLUDE, token.REQUIRE, token.INCLUDE_ONCE, token.REQUIRE_ONCE:
		return p.parseIncludeStatement()
	case token.NAMESPACE:
		return p.parseNamespaceStatement()
	case token.USE:
		return p.parseUseStatement()
	case token.PUBLIC, token.PRIVATE, token.PROTECTED:
		p.nextToken() // consume visibility modifier
		if p.curTokenIs(token.FUNCTION) {
			return p.parseFunctionStatement()
		}
		return nil
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseInlineHTMLStatement() *ast.InlineHTMLStatement {
	return &ast.InlineHTMLStatement{Token: p.curToken, Content: p.curToken.Literal}
}

func (p *Parser) parseEchoStatement() *ast.EchoStatement {
	stmt := &ast.EchoStatement{Token: p.curToken}
	p.nextToken() // consume 'echo'

	stmt.Expressions = []ast.Expression{}
	stmt.Expressions = append(stmt.Expressions, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // consume expression
		p.nextToken() // consume ','
		stmt.Expressions = append(stmt.Expressions, p.parseExpression(LOWEST))
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	} else if p.peekTokenIs(token.T_CLOSE_TAG) {
		// implicit semicolon before close tag
	} else {
		p.errors = append(p.errors, fmt.Sprintf("expected semicolon or close tag, got %s at line %d", p.peekToken.Type, p.peekToken.Pos.Line))
	}

	return stmt
}

func (p *Parser) parseIfStatement() *ast.IfStatement {
	stmt := &ast.IfStatement{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken() // consume '('
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	p.nextToken() // consume ')'

	if p.curTokenIs(token.LBRACE) {
		stmt.Consequence = p.parseBlockStatement()
	} else {
		singleStmt := p.parseStatement()
		if singleStmt != nil {
			stmt.Consequence = &ast.BlockStatement{
				Token:      token.Token{Type: token.LBRACE, Literal: "{"},
				Statements: []ast.Statement{singleStmt},
			}
		}
	}

	if p.peekTokenIs(token.ELSE) {
		p.nextToken() // consume consequence end
		p.nextToken() // consume ELSE

		if p.curTokenIs(token.LBRACE) {
			stmt.Alternative = p.parseBlockStatement()
		} else {
			stmt.Alternative = p.parseStatement()
		}
	}

	return stmt
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken() // consume '{'

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
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

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseVariable() ast.Expression {
	return &ast.Variable{Token: p.curToken, Value: p.curToken.Literal}
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

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}
	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("could not parse %q as integer", p.curToken.Literal))
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("could not parse %q as float", p.curToken.Literal))
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BooleanLiteral{Token: p.curToken, Value: p.curToken.Type == token.TRUE}
}

func (p *Parser) parseNullLiteral() ast.Expression {
	return &ast.NullLiteral{Token: p.curToken}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken() // consume '('
	exp := p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return exp
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}
	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)
	return expression
}

func (p *Parser) parseAssignExpression(left ast.Expression) ast.Expression {
	if _, ok := left.(*ast.Variable); !ok {
		if _, ok := left.(*ast.PropertyExpression); !ok {
			if _, ok := left.(*ast.IndexExpression); !ok {
				p.errors = append(p.errors, fmt.Sprintf("left side of assignment must be a variable, property, or index, got %T", left))
				return nil
			}
		}
	}

	expression := &ast.AssignExpression{
		Token: p.curToken,
		Left:  left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Value = p.parseExpression(precedence - 1)

	return expression
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
	}
	p.peekError(t)
	return false
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found at line %d, col %d", t, p.curToken.Pos.Line, p.curToken.Pos.Column)
	p.errors = append(p.errors, msg)
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

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}
	p.nextToken() // consume 'return'
	
	if !p.curTokenIs(token.SEMICOLON) && !p.curTokenIs(token.T_CLOSE_TAG) {
		stmt.ReturnValue = p.parseExpression(LOWEST)
	}
	
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseFunctionStatement() *ast.FunctionStatement {
	stmt := &ast.FunctionStatement{Token: p.curToken}
	if !p.expectPeek(token.T_IDENTIFIER) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	stmt.Parameters = p.parseFunctionParameters()
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseFunctionParameters() []*ast.Parameter {
	parameters := []*ast.Parameter{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return parameters
	}

	p.nextToken() // consume LPAREN

	param := p.parseParameter()
	parameters = append(parameters, param)

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		param := p.parseParameter()
		parameters = append(parameters, param)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return parameters
}

func (p *Parser) parseParameter() *ast.Parameter {
	param := &ast.Parameter{Token: p.curToken}
	if !p.curTokenIs(token.T_VARIABLE) {
		return nil
	}
	param.Var = &ast.Variable{Token: p.curToken, Value: p.curToken.Literal}
	
	if p.peekTokenIs(token.ASSIGN) {
		p.nextToken() // consume variable
		p.nextToken() // consume '='
		param.DefaultValue = p.parseExpression(LOWEST)
	}
	return param
}

func (p *Parser) parseClassStatement() *ast.ClassStatement {
	stmt := &ast.ClassStatement{Token: p.curToken}
	if !p.expectPeek(token.T_IDENTIFIER) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	p.nextToken() // consume '{'
	
	stmt.Methods = []*ast.FunctionStatement{}
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.PUBLIC) || p.curTokenIs(token.PRIVATE) || p.curTokenIs(token.PROTECTED) {
			p.nextToken()
		}
		if p.curTokenIs(token.FUNCTION) {
			fn := p.parseFunctionStatement()
			if fn != nil {
				stmt.Methods = append(stmt.Methods, fn)
			}
			p.nextToken()
		} else {
			p.nextToken()
		}
	}
	return stmt
}

func (p *Parser) parseNewExpression() ast.Expression {
	exp := &ast.NewExpression{Token: p.curToken}
	if !p.expectPeek(token.T_IDENTIFIER) {
		return nil
	}
	exp.Class = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if p.peekTokenIs(token.LPAREN) {
		p.nextToken() // consume identifier
		exp.Arguments = p.parseCallArguments()
	} else {
		exp.Arguments = []ast.Expression{}
	}
	return exp
}

func (p *Parser) parseCallArguments() []ast.Expression {
	args := []ast.Expression{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return args
	}

	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return args
}

func (p *Parser) parseCallExpression(left ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: left}
	exp.Arguments = p.parseCallArguments()
	return exp
}

func (p *Parser) parseObjectAccessExpression(left ast.Expression) ast.Expression {
	tok := p.curToken // '->'
	if !p.expectPeek(token.T_IDENTIFIER) {
		return nil
	}
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if p.peekTokenIs(token.LPAREN) {
		p.nextToken() // consume identifier
		exp := &ast.MethodCallExpression{
			Token:     tok,
			Object:    left,
			Method:    ident,
			Arguments: p.parseCallArguments(),
		}
		return exp
	}
	
	return &ast.PropertyExpression{
		Token:    tok,
		Object:   left,
		Property: ident,
	}
}

func (p *Parser) parseTernaryExpression(left ast.Expression) ast.Expression {
	exp := &ast.TernaryExpression{
		Token:     p.curToken,
		Condition: left,
	}
	p.nextToken() // consume '?'
	exp.Consequence = p.parseExpression(LOWEST)
	if !p.expectPeek(token.COLON) {
		return nil
	}
	p.nextToken() // consume ':'
	exp.Alternative = p.parseExpression(LOWEST)
	return exp
}

func (p *Parser) parseSwitchStatement() *ast.SwitchStatement {
	stmt := &ast.SwitchStatement{Token: p.curToken}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken() // consume '('
	stmt.Expr = p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	p.nextToken() // consume '{'
	
	stmt.Cases = []*ast.CaseStatement{}
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.CASE) {
			caseStmt := &ast.CaseStatement{Token: p.curToken}
			p.nextToken() // consume 'case'
			caseStmt.Value = p.parseExpression(LOWEST)
			if !p.expectPeek(token.COLON) {
				return nil
			}
			p.nextToken() // consume ':'
			
			caseStmt.Body = &ast.BlockStatement{Token: token.Token{Type: token.LBRACE, Literal: "{"}}
			caseStmt.Body.Statements = []ast.Statement{}
			for !p.curTokenIs(token.CASE) && !p.curTokenIs(token.DEFAULT) && !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
				if p.curTokenIs(token.BREAK) {
					p.nextToken() // consume break
					if p.curTokenIs(token.SEMICOLON) {
						p.nextToken()
					}
					continue
				}
				s := p.parseStatement()
				if s != nil {
					caseStmt.Body.Statements = append(caseStmt.Body.Statements, s)
				}
				p.nextToken()
			}
			stmt.Cases = append(stmt.Cases, caseStmt)
		} else if p.curTokenIs(token.DEFAULT) {
			if !p.expectPeek(token.COLON) {
				return nil
			}
			p.nextToken() // consume ':'
			stmt.Default = &ast.BlockStatement{Token: token.Token{Type: token.LBRACE, Literal: "{"}}
			stmt.Default.Statements = []ast.Statement{}
			for !p.curTokenIs(token.CASE) && !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
				if p.curTokenIs(token.BREAK) {
					p.nextToken() // consume break
					if p.curTokenIs(token.SEMICOLON) {
						p.nextToken()
					}
					continue
				}
				s := p.parseStatement()
				if s != nil {
					stmt.Default.Statements = append(stmt.Default.Statements, s)
				}
				p.nextToken()
			}
		} else {
			p.nextToken()
		}
	}
	return stmt
}

func (p *Parser) parseWhileStatement() *ast.WhileStatement {
	stmt := &ast.WhileStatement{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseDoWhileStatement() *ast.DoWhileStatement {
	stmt := &ast.DoWhileStatement{Token: p.curToken}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	if !p.expectPeek(token.WHILE) {
		return nil
	}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseForStatement() *ast.ForStatement {
	stmt := &ast.ForStatement{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()
	if !p.curTokenIs(token.SEMICOLON) {
		stmt.Init = p.parseExpression(LOWEST)
		if !p.expectPeek(token.SEMICOLON) {
			return nil
		}
	}

	p.nextToken()
	if !p.curTokenIs(token.SEMICOLON) {
		stmt.Condition = p.parseExpression(LOWEST)
		if !p.expectPeek(token.SEMICOLON) {
			return nil
		}
	}

	p.nextToken()
	if !p.curTokenIs(token.RPAREN) {
		stmt.Post = p.parseExpression(LOWEST)
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

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

func (p *Parser) parsePostExpression(left ast.Expression) ast.Expression {
	return &ast.PostExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}
}

func (p *Parser) parseIncludeStatement() *ast.IncludeStatement {
	stmt := &ast.IncludeStatement{Token: p.curToken}
	p.nextToken() // consume include/require
	stmt.Expression = p.parseExpression(LOWEST)
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseNamespaceStatement() *ast.NamespaceStatement {
	stmt := &ast.NamespaceStatement{Token: p.curToken}
	if !p.expectPeek(token.T_IDENTIFIER) {
		return nil
	}
	stmt.Name = p.curToken
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseUseStatement() *ast.UseStatement {
	stmt := &ast.UseStatement{Token: p.curToken}
	if !p.expectPeek(token.T_IDENTIFIER) {
		return nil
	}
	stmt.Name = p.curToken

	parts := strings.Split(stmt.Name.Literal, "\\")
	defaultAlias := parts[len(parts)-1]
	stmt.Alias = token.Token{Type: token.T_IDENTIFIER, Literal: defaultAlias}

	if p.peekTokenIs(token.AS) {
		p.nextToken() // consume 'as'
		if !p.expectPeek(token.T_IDENTIFIER) {
			return nil
		}
		stmt.Alias = p.curToken
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseFunctionExpression() ast.Expression {
	tok := p.curToken
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	params := p.parseFunctionParameters()
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	body := p.parseBlockStatement()
	return &ast.FunctionExpression{Token: tok, Parameters: params, Body: body}
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	lit := &ast.ArrayLiteral{Token: p.curToken}
	lit.Elements = p.parseArrayElements()
	return lit
}

func (p *Parser) parseArrayElements() []*ast.ArrayElement {
	elements := []*ast.ArrayElement{}
	if p.peekTokenIs(token.RBRACKET) {
		p.nextToken()
		return elements
	}
	
	p.nextToken() // advance to first element

	for {
		val1 := p.parseExpression(LOWEST)
		
		var elem *ast.ArrayElement
		if p.peekTokenIs(token.DOUBLE_ARROW) {
			p.nextToken() // consume val1, curToken is '=>'
			p.nextToken() // consume '=>'
			val2 := p.parseExpression(LOWEST)
			elem = &ast.ArrayElement{Key: val1, Value: val2}
		} else {
			elem = &ast.ArrayElement{Key: nil, Value: val1}
		}
		elements = append(elements, elem)

		if p.peekTokenIs(token.COMMA) {
			p.nextToken()
			if p.peekTokenIs(token.RBRACKET) {
				p.nextToken()
				break
			}
			p.nextToken()
		} else if p.peekTokenIs(token.RBRACKET) {
			p.nextToken()
			break
		} else {
			break
		}
	}
	return elements
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}
	p.nextToken() // consume '['
	
	if p.curTokenIs(token.RBRACKET) {
		exp.Index = nil
		return exp
	}
	
	exp.Index = p.parseExpression(LOWEST)
	
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	
	return exp
}

