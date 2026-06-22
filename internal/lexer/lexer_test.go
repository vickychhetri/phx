package lexer

import (
	"testing"
	"phx/internal/token"
)

func TestNextToken(t *testing.T) {
	input := `<html>
<?php
// This is a comment
$name = "Vicky";
$x = 10;
$y = 20.5;
/* Multi-line
   comment */
if ($x === $y) {
    echo "equal";
} else {
    echo 'not equal';
}
?>
Done!
`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.T_INLINE_HTML, "<html>\n"},
		{token.T_OPEN_TAG, "<?php"},
		{token.T_VARIABLE, "$name"},
		{token.ASSIGN, "="},
		{token.T_DOUBLE_QUOTED_STRING, "Vicky"},
		{token.SEMICOLON, ";"},
		{token.T_VARIABLE, "$x"},
		{token.ASSIGN, "="},
		{token.T_LNUMBER, "10"},
		{token.SEMICOLON, ";"},
		{token.T_VARIABLE, "$y"},
		{token.ASSIGN, "="},
		{token.T_DNUMBER, "20.5"},
		{token.SEMICOLON, ";"},
		{token.IF, "if"},
		{token.LPAREN, "("},
		{token.T_VARIABLE, "$x"},
		{token.IDENTICAL, "==="},
		{token.T_VARIABLE, "$y"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.ECHO, "echo"},
		{token.T_DOUBLE_QUOTED_STRING, "equal"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.ELSE, "else"},
		{token.LBRACE, "{"},
		{token.ECHO, "echo"},
		{token.T_CONSTANT_ENCAPSED_STRING, "not equal"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.T_CLOSE_TAG, "?>"},
		{token.T_INLINE_HTML, "\nDone!\n"},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal: %q)",
				i, tt.expectedType, tok.Type, tok.Literal)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}
