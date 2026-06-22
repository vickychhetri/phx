package token

type TokenType string

type Position struct {
	Line   int
	Column int
	Offset int
}

type Token struct {
	Type    TokenType
	Literal string
	Pos     Position
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers & variables
	T_IDENTIFIER = "T_IDENTIFIER" // e.g. foo
	T_VARIABLE   = "T_VARIABLE"   // e.g. $foo

	// Literals
	T_LNUMBER                  = "T_LNUMBER"                  // integer
	T_DNUMBER                  = "T_DNUMBER"                  // float
	T_CONSTANT_ENCAPSED_STRING = "T_CONSTANT_ENCAPSED_STRING" // string literal
	T_DOUBLE_QUOTED_STRING     = "T_DOUBLE_QUOTED_STRING"     // double quoted string literal
	T_INLINE_HTML              = "T_INLINE_HTML"              // raw HTML outside php tags

	// Operators
	ASSIGN        = "="
	PLUS          = "+"
	MINUS         = "-"
	MUL           = "*"
	DIV           = "/"
	MOD           = "%"
	CONCAT        = "."
	EQUAL         = "=="
	IDENTICAL     = "==="
	NOT_EQUAL     = "!="
	NOT_IDENTICAL = "!=="
	LT            = "<"
	GT            = ">"
	LTE           = "<="
	GTE           = ">="

	BANG = "!"

	ADD_ASSIGN = "+="
	SUB_ASSIGN = "-="

	INC = "++"
	DEC = "--"

	OBJECT_OPERATOR = "->"
	DOUBLE_ARROW    = "=>"
	QUESTION        = "?"
	COLON           = ":"

	// Delimiters
	SEMICOLON = ";"
	COMMA     = ","
	LPAREN    = "("
	RPAREN    = ")"
	LBRACE    = "{"
	RBRACE    = "}"
	LBRACKET  = "["
	RBRACKET  = "]"

	// PHP tags
	T_OPEN_TAG  = "T_OPEN_TAG"  // <?php or <?
	T_CLOSE_TAG = "T_CLOSE_TAG" // ?>

	// Keywords
	ECHO     = "echo"
	IF       = "if"
	ELSE     = "else"
	ELSEIF   = "elseif"
	FUNCTION = "function"
	CLASS    = "class"
	RETURN   = "return"
	TRUE     = "true"
	FALSE    = "false"
	NULL     = "null"

	SWITCH    = "switch"
	CASE      = "case"
	DEFAULT   = "default"
	NEW       = "new"
	PUBLIC    = "public"
	PROTECTED = "protected"
	PRIVATE   = "private"

	WHILE    = "while"
	DO       = "do"
	FOR      = "for"
	FOREACH  = "foreach"
	AS       = "as"
	BREAK    = "break"
	CONTINUE = "continue"

	INCLUDE      = "include"
	REQUIRE      = "require"
	INCLUDE_ONCE = "include_once"
	REQUIRE_ONCE = "require_once"
	NAMESPACE    = "namespace"
	USE          = "use"
)

var keywords = map[string]TokenType{
	"echo":         ECHO,
	"if":           IF,
	"else":         ELSE,
	"elseif":       ELSEIF,
	"function":     FUNCTION,
	"class":        CLASS,
	"return":       RETURN,
	"true":         TRUE,
	"false":        FALSE,
	"null":         NULL,
	"switch":       SWITCH,
	"case":         CASE,
	"default":      DEFAULT,
	"new":          NEW,
	"public":       PUBLIC,
	"protected":    PROTECTED,
	"private":      PRIVATE,
	"while":        WHILE,
	"do":           DO,
	"for":          FOR,
	"foreach":      FOREACH,
	"as":           AS,
	"break":        BREAK,
	"continue":     CONTINUE,
	"include":      INCLUDE,
	"require":      REQUIRE,
	"include_once": INCLUDE_ONCE,
	"require_once": REQUIRE_ONCE,
	"namespace":    NAMESPACE,
	"use":          USE,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return T_IDENTIFIER
}

func NewToken(tokenType TokenType, ch byte, pos Position) Token {
	return Token{Type: tokenType, Literal: string(ch), Pos: pos}
}

