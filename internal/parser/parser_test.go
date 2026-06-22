package parser

import (
	"phx/internal/ast"
	"phx/internal/lexer"
	"testing"
)

func TestParseHTMLAndEcho(t *testing.T) {
	input := `Hello World<?php echo "Hello", $name; ?>`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 2 {
		t.Fatalf("program.Statements does not contain 2 statements. got=%d", len(program.Statements))
	}

	// 1. Inline HTML
	htmlStmt, ok := program.Statements[0].(*ast.InlineHTMLStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *ast.InlineHTMLStatement. got=%T", program.Statements[0])
	}
	if htmlStmt.Content != "Hello World" {
		t.Errorf("htmlStmt.Content not 'Hello World'. got=%q", htmlStmt.Content)
	}

	// 2. Echo
	echoStmt, ok := program.Statements[1].(*ast.EchoStatement)
	if !ok {
		t.Fatalf("program.Statements[1] is not *ast.EchoStatement. got=%T", program.Statements[1])
	}
	if len(echoStmt.Expressions) != 2 {
		t.Fatalf("echoStmt.Expressions does not contain 2 expressions. got=%d", len(echoStmt.Expressions))
	}
}

func TestAssignAndPrecedence(t *testing.T) {
	input := `<?php
$x = 10 + 20 * 30;
$y = ($x + 5) * 2.5;
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 2 {
		t.Fatalf("program.Statements does not contain 2 statements. got=%d", len(program.Statements))
	}

	// Stmt 1: $x = (10 + (20 * 30))
	stmt1, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("stmt1 is not *ast.ExpressionStatement. got=%T", program.Statements[0])
	}
	assign1, ok := stmt1.Expression.(*ast.AssignExpression)
	if !ok {
		t.Fatalf("expression is not *ast.AssignExpression. got=%T", stmt1.Expression)
	}
	variable1, ok := assign1.Left.(*ast.Variable)
	if !ok {
		t.Fatalf("assign1.Left is not *ast.Variable. got=%T", assign1.Left)
	}
	if variable1.Value != "$x" {
		t.Errorf("assign1.Left.Value is not '$x'. got=%q", variable1.Value)
	}

	expectedStr1 := "$x = (10 + (20 * 30))"
	if assign1.String() != expectedStr1 {
		t.Errorf("assign1.String() wrong. expected=%q, got=%q", expectedStr1, assign1.String())
	}

	// Stmt 2: $y = (($x + 5) * 2.5)
	stmt2, ok := program.Statements[1].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("stmt2 is not *ast.ExpressionStatement. got=%T", program.Statements[1])
	}
	assign2, ok := stmt2.Expression.(*ast.AssignExpression)
	if !ok {
		t.Fatalf("expression is not *ast.AssignExpression. got=%T", stmt2.Expression)
	}

	expectedStr2 := "$y = (($x + 5) * 2.5)"
	if assign2.String() != expectedStr2 {
		t.Errorf("assign2.String() wrong. expected=%q, got=%q", expectedStr2, assign2.String())
	}
}

func TestIfElseStatements(t *testing.T) {
	input := `<?php
if ($x === true) {
    echo "yes";
} else {
    echo "no";
}
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	ifStmt, ok := program.Statements[0].(*ast.IfStatement)
	if !ok {
		t.Fatalf("statement is not *ast.IfStatement. got=%T", program.Statements[0])
	}

	// Condition
	cond, ok := ifStmt.Condition.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("condition is not *ast.InfixExpression. got=%T", ifStmt.Condition)
	}
	if cond.Operator != "===" {
		t.Errorf("operator is not '==='. got=%q", cond.Operator)
	}

	// Consequence block
	if len(ifStmt.Consequence.Statements) != 1 {
		t.Fatalf("consequence block does not have 1 statement. got=%d", len(ifStmt.Consequence.Statements))
	}

	// Alternative block
	alt, ok := ifStmt.Alternative.(*ast.BlockStatement)
	if !ok {
		t.Fatalf("alternative is not *ast.BlockStatement. got=%T", ifStmt.Alternative)
	}
	if len(alt.Statements) != 1 {
		t.Fatalf("alternative block does not have 1 statement. got=%d", len(alt.Statements))
	}
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))
	for _, msg := range errors {
		t.Errorf("parser error: %q", msg)
	}
	t.FailNow()
}
