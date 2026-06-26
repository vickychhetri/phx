package lexer

import (
	"phx/internal/token"
)

type Lexer struct {
	input        string
	position     int  // current index in input (points to current char)
	readPosition int  // current reading index in input (after current char)
	ch           byte // current char under examination
	line         int
	col          int
	inPHPMode    bool // tracks if we are inside <?php ... ?>
}

func New(input string) *Lexer {
	l := &Lexer{
		input:     input,
		line:      1,
		col:       0,
		inPHPMode: false,
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	if l.ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) currentPos() token.Position {
	return token.Position{
		Line:   l.line,
		Column: l.col,
		Offset: l.position,
	}
}

func (l *Lexer) hasPrefix(prefix string) bool {
	if l.position+len(prefix) > len(l.input) {
		return false
	}
	return l.input[l.position:l.position+len(prefix)] == prefix
}

func (l *Lexer) readHTML() token.Token {
	startPos := l.position
	startLine := l.line
	startCol := l.col

	for l.ch != 0 {
		if l.ch == '<' && (l.hasPrefix("<?php") || l.hasPrefix("<?")) {
			break
		}
		l.readChar()
	}

	literal := l.input[startPos:l.position]
	return token.Token{
		Type:    token.T_INLINE_HTML,
		Literal: literal,
		Pos: token.Position{
			Line:   startLine,
			Column: startCol,
			Offset: startPos,
		},
	}
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) skipSingleLineComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

func (l *Lexer) skipMultiLineComment() {
	l.readChar() // consume '*'
	l.readChar() // consume next char
	for l.ch != 0 {
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar() // consume '*'
			l.readChar() // consume '/'
			break
		}
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() token.Token {
	startPos := l.currentPos()
	position := l.position
	isFloat := false
	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		l.readChar() // consume '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	literal := l.input[position:l.position]
	var tokType token.TokenType = token.T_LNUMBER
	if isFloat {
		tokType = token.T_DNUMBER
	}

	return token.Token{
		Type:    tokType,
		Literal: literal,
		Pos:     startPos,
	}
}

func (l *Lexer) readString(quote byte) token.Token {
	startPos := l.currentPos()
	l.readChar() // consume opening quote
	position := l.position

	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar() // skip escaped character
		}
		l.readChar()
	}

	literal := l.input[position:l.position]
	if l.ch == quote {
		l.readChar() // consume closing quote
	}

	var tokType token.TokenType = token.T_CONSTANT_ENCAPSED_STRING
	if quote == '"' {
		tokType = token.T_DOUBLE_QUOTED_STRING
	}

	return token.Token{
		Type:    tokType,
		Literal: literal,
		Pos:     startPos,
	}
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	if !l.inPHPMode {
		if l.ch == 0 {
			tok = token.Token{
				Type: token.EOF,
				Pos:  l.currentPos(),
			}
			return tok
		}

		if l.ch == '<' {
			if l.hasPrefix("<?php") {
				tok = token.Token{
					Type:    token.T_OPEN_TAG,
					Literal: "<?php",
					Pos:     l.currentPos(),
				}
				for i := 0; i < 5; i++ {
					l.readChar()
				}
				l.inPHPMode = true
				return tok
			}
			if l.hasPrefix("<?") {
				tok = token.Token{
					Type:    token.T_OPEN_TAG,
					Literal: "<?",
					Pos:     l.currentPos(),
				}
				for i := 0; i < 2; i++ {
					l.readChar()
				}
				l.inPHPMode = true
				return tok
			}
		}

		return l.readHTML()
	}

	l.skipWhitespace()

	if l.ch == 0 {
		tok = token.Token{
			Type: token.EOF,
			Pos:  l.currentPos(),
		}
		return tok
	}

	if l.ch == '?' && l.peekChar() == '>' {
		tok = token.Token{
			Type:    token.T_CLOSE_TAG,
			Literal: "?>",
			Pos:     l.currentPos(),
		}
		l.readChar() // consume '?'
		l.readChar() // consume '>'
		l.inPHPMode = false
		return tok
	}

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			startPos := l.currentPos()
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = token.Token{Type: token.IDENTICAL, Literal: "===", Pos: startPos}
			} else {
				tok = token.Token{Type: token.EQUAL, Literal: "==", Pos: startPos}
			}
		} else if l.peekChar() == '>' {
			startPos := l.currentPos()
			l.readChar()
			tok = token.Token{Type: token.DOUBLE_ARROW, Literal: "=>", Pos: startPos}
		} else {
			tok = token.NewToken(token.ASSIGN, l.ch, l.currentPos())
		}
	case '+':
		if l.peekChar() == '=' {
			startPos := l.currentPos()
			l.readChar()
			tok = token.Token{Type: token.ADD_ASSIGN, Literal: "+=", Pos: startPos}
		} else if l.peekChar() == '+' {
			startPos := l.currentPos()
			l.readChar()
			tok = token.Token{Type: token.INC, Literal: "++", Pos: startPos}
		} else {
			tok = token.NewToken(token.PLUS, l.ch, l.currentPos())
		}
	case '-':
		if l.peekChar() == '=' {
			startPos := l.currentPos()
			l.readChar()
			tok = token.Token{Type: token.SUB_ASSIGN, Literal: "-=", Pos: startPos}
		} else if l.peekChar() == '-' {
			startPos := l.currentPos()
			l.readChar()
			tok = token.Token{Type: token.DEC, Literal: "--", Pos: startPos}
		} else if l.peekChar() == '>' {
			startPos := l.currentPos()
			l.readChar() // consume '-'
			tok = token.Token{Type: token.OBJECT_OPERATOR, Literal: "->", Pos: startPos}
		} else {
			tok = token.NewToken(token.MINUS, l.ch, l.currentPos())
		}
	case '*':
		tok = token.NewToken(token.MUL, l.ch, l.currentPos())
	case '/':
		if l.peekChar() == '/' {
			l.skipSingleLineComment()
			return l.NextToken()
		} else if l.peekChar() == '*' {
			l.skipMultiLineComment()
			return l.NextToken()
		}
		tok = token.NewToken(token.DIV, l.ch, l.currentPos())
	case '%':
		tok = token.NewToken(token.MOD, l.ch, l.currentPos())
	case '.':
		if l.peekChar() == '=' {
			startPos := l.currentPos()
			l.readChar()
			tok = token.Token{Type: token.CONCAT_ASSIGN, Literal: ".=", Pos: startPos}
		} else {
			tok = token.NewToken(token.CONCAT, l.ch, l.currentPos())
		}
	case ';':
		tok = token.NewToken(token.SEMICOLON, l.ch, l.currentPos())
	case ',':
		tok = token.NewToken(token.COMMA, l.ch, l.currentPos())
	case '(':
		tok = token.NewToken(token.LPAREN, l.ch, l.currentPos())
	case ')':
		tok = token.NewToken(token.RPAREN, l.ch, l.currentPos())
	case '{':
		tok = token.NewToken(token.LBRACE, l.ch, l.currentPos())
	case '}':
		tok = token.NewToken(token.RBRACE, l.ch, l.currentPos())
	case '[':
		tok = token.NewToken(token.LBRACKET, l.ch, l.currentPos())
	case ']':
		tok = token.NewToken(token.RBRACKET, l.ch, l.currentPos())
	case '<':
		if l.peekChar() == '=' {
			startPos := l.currentPos()
			l.readChar()
			tok = token.Token{Type: token.LTE, Literal: "<=", Pos: startPos}
		} else {
			tok = token.NewToken(token.LT, l.ch, l.currentPos())
		}
	case '>':
		if l.peekChar() == '=' {
			startPos := l.currentPos()
			l.readChar()
			tok = token.Token{Type: token.GTE, Literal: ">=", Pos: startPos}
		} else {
			tok = token.NewToken(token.GT, l.ch, l.currentPos())
		}
	case '!':
		if l.peekChar() == '=' {
			startPos := l.currentPos()
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = token.Token{Type: token.NOT_IDENTICAL, Literal: "!==", Pos: startPos}
			} else {
				tok = token.Token{Type: token.NOT_EQUAL, Literal: "!=", Pos: startPos}
			}
		} else {
			tok = token.NewToken(token.BANG, l.ch, l.currentPos())
		}
	case '"', '\'':
		tok = l.readString(l.ch)
		return tok
	case '&':
		if l.peekChar() == '&' {
			startPos := l.currentPos()
			l.readChar()
			tok = token.Token{Type: token.AND, Literal: "&&", Pos: startPos}
		} else {
			tok = token.NewToken(token.BITWISE_AND, l.ch, l.currentPos())
		}
	case '|':
		if l.peekChar() == '|' {
			startPos := l.currentPos()
			l.readChar()
			tok = token.Token{Type: token.OR, Literal: "||", Pos: startPos}
		} else {
			tok = token.NewToken(token.BITWISE_OR, l.ch, l.currentPos())
		}
	case '@':
		tok = token.NewToken(token.AT, l.ch, l.currentPos())
	case '?':
		if l.peekChar() == '?' {
			startPos := l.currentPos()
			l.readChar()
			tok = token.Token{Type: token.COALESCE, Literal: "??", Pos: startPos}
		} else {
			tok = token.NewToken(token.QUESTION, l.ch, l.currentPos())
		}
	case ':':
		if l.peekChar() == ':' {
			startPos := l.currentPos()
			l.readChar()
			tok = token.Token{Type: token.DOUBLE_COLON, Literal: "::", Pos: startPos}
		} else {
			tok = token.NewToken(token.COLON, l.ch, l.currentPos())
		}
	case '$':
		if isLetter(l.peekChar()) {
			startPos := l.currentPos()
			l.readChar() // consume '$'
			ident := l.readIdentifier()
			tok = token.Token{
				Type:    token.T_VARIABLE,
				Literal: "$" + ident,
				Pos:     startPos,
			}
			return tok
		}
		tok = token.NewToken(token.ILLEGAL, l.ch, l.currentPos())
	default:
		if isLetter(l.ch) {
			startPos := l.currentPos()
			ident := l.readIdentifier()
			tok = token.Token{
				Type:    token.LookupIdent(ident),
				Literal: ident,
				Pos:     startPos,
			}
			return tok
		} else if isDigit(l.ch) {
			return l.readNumber()
		} else {
			tok = token.NewToken(token.ILLEGAL, l.ch, l.currentPos())
		}
	}

	l.readChar()
	return tok
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch == '\\'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
