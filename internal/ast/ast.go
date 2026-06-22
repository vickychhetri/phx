package ast

import (
	"bytes"
	"phx/internal/token"
	"strings"
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

// Program is the root node of the AST
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var out bytes.Buffer
	for _, s := range p.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

// InlineHTMLStatement represents HTML content outside of <?php ?> tags
type InlineHTMLStatement struct {
	Token   token.Token
	Content string
}

func (ih *InlineHTMLStatement) statementNode()       {}
func (ih *InlineHTMLStatement) TokenLiteral() string { return ih.Token.Literal }
func (ih *InlineHTMLStatement) String() string       { return ih.Content }

// EchoStatement represents PHP 'echo' statement
type EchoStatement struct {
	Token       token.Token // The 'echo' token
	Expressions []Expression
}

func (es *EchoStatement) statementNode()       {}
func (es *EchoStatement) TokenLiteral() string { return es.Token.Literal }
func (es *EchoStatement) String() string {
	var out bytes.Buffer
	out.WriteString(es.TokenLiteral() + " ")
	exprStrings := []string{}
	for _, expr := range es.Expressions {
		exprStrings = append(exprStrings, expr.String())
	}
	out.WriteString(strings.Join(exprStrings, ", "))
	out.WriteString(";")
	return out.String()
}

// ExpressionStatement represents a statement consisting of a single expression
type ExpressionStatement struct {
	Token      token.Token // the first token of the expression
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String() + ";"
	}
	return ""
}

// BlockStatement represents a block of statements enclosed in {}
type BlockStatement struct {
	Token      token.Token // '{'
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out bytes.Buffer
	out.WriteString("{ ")
	for _, s := range bs.Statements {
		out.WriteString(s.String() + " ")
	}
	out.WriteString("}")
	return out.String()
}

// IfStatement represents 'if (condition) consequence else alternative'
type IfStatement struct {
	Token       token.Token // 'if'
	Condition   Expression
	Consequence *BlockStatement
	Alternative Statement // Can be BlockStatement, IfStatement, or nil
}

func (is *IfStatement) statementNode()       {}
func (is *IfStatement) TokenLiteral() string { return is.Token.Literal }
func (is *IfStatement) String() string {
	var out bytes.Buffer
	out.WriteString("if (")
	out.WriteString(is.Condition.String())
	out.WriteString(") ")
	out.WriteString(is.Consequence.String())
	if is.Alternative != nil {
		out.WriteString(" else ")
		out.WriteString(is.Alternative.String())
	}
	return out.String()
}

// Variable represents PHP variables (e.g. $name)
type Variable struct {
	Token token.Token // T_VARIABLE
	Value string      // e.g. "$name"
}

func (v *Variable) expressionNode()      {}
func (v *Variable) TokenLiteral() string { return v.Token.Literal }
func (v *Variable) String() string       { return v.Value }

// Identifier represents keywords or names (e.g. function name)
type Identifier struct {
	Token token.Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// IntegerLiteral represents an integer (e.g. 42)
type IntegerLiteral struct {
	Token token.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// FloatLiteral represents a float (e.g. 3.14)
type FloatLiteral struct {
	Token token.Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

// StringLiteral represents a string constant
type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return "\"" + sl.Value + "\"" }

// BooleanLiteral represents true or false
type BooleanLiteral struct {
	Token token.Token
	Value bool
}

func (bl *BooleanLiteral) expressionNode()      {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Literal }
func (bl *BooleanLiteral) String() string       { return bl.Token.Literal }

// NullLiteral represents null
type NullLiteral struct {
	Token token.Token
}

func (nl *NullLiteral) expressionNode()      {}
func (nl *NullLiteral) TokenLiteral() string { return nl.Token.Literal }
func (nl *NullLiteral) String() string       { return "null" }

// AssignExpression represents variable assignment ($x = 10)
type AssignExpression struct {
	Token token.Token // '=' or '+=' or '-='
	Left  Expression
	Value Expression
}

func (ae *AssignExpression) expressionNode()      {}
func (ae *AssignExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AssignExpression) String() string {
	return ae.Left.String() + " " + ae.Token.Literal + " " + ae.Value.String()
}

// InfixExpression represents binary operations (e.g. 5 + 10)
type InfixExpression struct {
	Token    token.Token // The operator token (e.g. +)
	Left     Expression
	Operator string
	Right    Expression
}

func (oe *InfixExpression) expressionNode()      {}
func (oe *InfixExpression) TokenLiteral() string { return oe.Token.Literal }
func (oe *InfixExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(oe.Left.String())
	out.WriteString(" " + oe.Operator + " ")
	out.WriteString(oe.Right.String())
	out.WriteString(")")
	return out.String()
}

// ReturnStatement represents 'return <expression>;'
type ReturnStatement struct {
	Token       token.Token // 'return'
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
	out.WriteString(";")
	return out.String()
}

type Parameter struct {
	Token        token.Token // variable token
	Var          *Variable
	DefaultValue Expression // nil if no default
}

func (p *Parameter) String() string {
	if p.DefaultValue != nil {
		return p.Var.String() + " = " + p.DefaultValue.String()
	}
	return p.Var.String()
}

// FunctionStatement represents function definition
type FunctionStatement struct {
	Token      token.Token // 'function'
	Name       *Identifier
	Parameters []*Parameter
	Body       *BlockStatement
}

func (fs *FunctionStatement) statementNode()       {}
func (fs *FunctionStatement) expressionNode()      {}
func (fs *FunctionStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *FunctionStatement) String() string {
	var out bytes.Buffer
	out.WriteString(fs.TokenLiteral() + " " + fs.Name.String() + "(")
	params := []string{}
	for _, p := range fs.Parameters {
		params = append(params, p.String())
	}
	out.WriteString(strings.Join(params, ", ") + ") ")
	out.WriteString(fs.Body.String())
	return out.String()
}

// FunctionExpression represents an anonymous function/closure
type FunctionExpression struct {
	Token      token.Token // 'function'
	Parameters []*Parameter
	UseVars    []*Variable  // variables captured via `use ($a, $b, ...)`
	Body       *BlockStatement
}

func (fe *FunctionExpression) expressionNode()      {}
func (fe *FunctionExpression) TokenLiteral() string { return fe.Token.Literal }
func (fe *FunctionExpression) String() string {
	var out bytes.Buffer
	out.WriteString(fe.TokenLiteral() + " (")
	params := []string{}
	for _, p := range fe.Parameters {
		params = append(params, p.String())
	}
	out.WriteString(strings.Join(params, ", ") + ") ")
	out.WriteString(fe.Body.String())
	return out.String()
}

// ClassStatement represents class definition
type ClassStatement struct {
	Token   token.Token // 'class'
	Name    *Identifier
	Methods []*FunctionStatement
}

func (cs *ClassStatement) statementNode()       {}
func (cs *ClassStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ClassStatement) String() string {
	var out bytes.Buffer
	out.WriteString(cs.TokenLiteral() + " " + cs.Name.String() + " { ")
	for _, m := range cs.Methods {
		out.WriteString(m.String() + " ")
	}
	out.WriteString("}")
	return out.String()
}

// NewExpression represents 'new ClassName(...)'
type NewExpression struct {
	Token     token.Token // 'new'
	Class     *Identifier
	Arguments []Expression
}

func (ne *NewExpression) expressionNode()      {}
func (ne *NewExpression) TokenLiteral() string { return ne.Token.Literal }
func (ne *NewExpression) String() string {
	var out bytes.Buffer
	out.WriteString("new " + ne.Class.String() + "(")
	args := []string{}
	for _, a := range ne.Arguments {
		args = append(args, a.String())
	}
	out.WriteString(strings.Join(args, ", ") + ")")
	return out.String()
}

// MethodCallExpression represents '$obj->method(args)'
type MethodCallExpression struct {
	Token     token.Token // '->'
	Object    Expression
	Method    *Identifier
	Arguments []Expression
}

func (mce *MethodCallExpression) expressionNode()      {}
func (mce *MethodCallExpression) TokenLiteral() string { return mce.Token.Literal }
func (mce *MethodCallExpression) String() string {
	var out bytes.Buffer
	out.WriteString(mce.Object.String() + "->" + mce.Method.String() + "(")
	args := []string{}
	for _, a := range mce.Arguments {
		args = append(args, a.String())
	}
	out.WriteString(strings.Join(args, ", ") + ")")
	return out.String()
}

// CallExpression represents 'function(args)'
type CallExpression struct {
	Token     token.Token // '('
	Function  Expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var out bytes.Buffer
	out.WriteString(ce.Function.String() + "(")
	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}
	out.WriteString(strings.Join(args, ", ") + ")")
	return out.String()
}

// SwitchStatement represents switch-case
type SwitchStatement struct {
	Token   token.Token // 'switch'
	Expr    Expression
	Cases   []*CaseStatement
	Default *BlockStatement
}

func (ss *SwitchStatement) statementNode()       {}
func (ss *SwitchStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *SwitchStatement) String() string {
	var out bytes.Buffer
	out.WriteString("switch (" + ss.Expr.String() + ") { ")
	for _, c := range ss.Cases {
		out.WriteString(c.String() + " ")
	}
	if ss.Default != nil {
		out.WriteString("default: " + ss.Default.String() + " ")
	}
	out.WriteString("}")
	return out.String()
}

// CaseStatement represents a case option
type CaseStatement struct {
	Token token.Token // 'case'
	Value Expression
	Body  *BlockStatement
}

func (cs *CaseStatement) statementNode()       {}
func (cs *CaseStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *CaseStatement) String() string {
	return "case " + cs.Value.String() + ": " + cs.Body.String()
}

// TernaryExpression represents 'cond ? expr1 : expr2'
type TernaryExpression struct {
	Token       token.Token // '?'
	Condition   Expression
	Consequence Expression
	Alternative Expression
}

func (te *TernaryExpression) expressionNode()      {}
func (te *TernaryExpression) TokenLiteral() string { return te.Token.Literal }
func (te *TernaryExpression) String() string {
	return te.Condition.String() + " ? " + te.Consequence.String() + " : " + te.Alternative.String()
}

// PropertyExpression represents member property access (e.g. $obj->prop)
type PropertyExpression struct {
	Token    token.Token // '->'
	Object   Expression
	Property *Identifier
}

func (pe *PropertyExpression) expressionNode()      {}
func (pe *PropertyExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PropertyExpression) String() string {
	return pe.Object.String() + "->" + pe.Property.String()
}

// PrefixExpression represents prefix unary operations (e.g. -5, !$x)
type PrefixExpression struct {
	Token    token.Token // The prefix token (e.g. -, !)
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())
	out.WriteString(")")
	return out.String()
}

// WhileStatement represents 'while (cond) { body }'
type WhileStatement struct {
	Token     token.Token // 'while'
	Condition Expression
	Body      *BlockStatement
}

func (ws *WhileStatement) statementNode()       {}
func (ws *WhileStatement) TokenLiteral() string { return ws.Token.Literal }
func (ws *WhileStatement) String() string {
	return "while (" + ws.Condition.String() + ") " + ws.Body.String()
}

// DoWhileStatement represents 'do { body } while (cond);'
type DoWhileStatement struct {
	Token     token.Token // 'do'
	Condition Expression
	Body      *BlockStatement
}

func (dws *DoWhileStatement) statementNode()       {}
func (dws *DoWhileStatement) TokenLiteral() string { return dws.Token.Literal }
func (dws *DoWhileStatement) String() string {
	return "do " + dws.Body.String() + " while (" + dws.Condition.String() + ");"
}

// ForStatement represents 'for (init; cond; post) { body }'
type ForStatement struct {
	Token     token.Token // 'for'
	Init      Expression  // Can be nil
	Condition Expression  // Can be nil
	Post      Expression  // Can be nil
	Body      *BlockStatement
}

func (fs *ForStatement) statementNode()       {}
func (fs *ForStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *ForStatement) String() string {
	var out bytes.Buffer
	out.WriteString("for (")
	if fs.Init != nil {
		out.WriteString(fs.Init.String())
	}
	out.WriteString("; ")
	if fs.Condition != nil {
		out.WriteString(fs.Condition.String())
	}
	out.WriteString("; ")
	if fs.Post != nil {
		out.WriteString(fs.Post.String())
	}
	out.WriteString(") ")
	out.WriteString(fs.Body.String())
	return out.String()
}

// BreakStatement represents 'break;'
type BreakStatement struct {
	Token token.Token // 'break'
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) String() string { return "break;" }

// ContinueStatement represents 'continue;'
type ContinueStatement struct {
	Token token.Token // 'continue'
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) String() string { return "continue;" }

// PostExpression represents postfix unary operations (e.g. $i++, $i--)
type PostExpression struct {
	Token    token.Token // The postfix token (e.g. ++, --)
	Operator string
	Left     Expression
}

func (pe *PostExpression) expressionNode()      {}
func (pe *PostExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PostExpression) String() string {
	return "(" + pe.Left.String() + pe.Operator + ")"
}

// IncludeStatement represents 'include ...;' or 'require ...;'
type IncludeStatement struct {
	Token      token.Token // 'include', 'require', etc.
	Expression Expression  // File path expression
}

func (is *IncludeStatement) statementNode()       {}
func (is *IncludeStatement) TokenLiteral() string { return is.Token.Literal }
func (is *IncludeStatement) String() string {
	return is.Token.Literal + " " + is.Expression.String() + ";"
}

// NamespaceStatement represents 'namespace MyPackage;'
type NamespaceStatement struct {
	Token token.Token // 'namespace'
	Name  token.Token // Name token (including backslashes if any)
}

func (ns *NamespaceStatement) statementNode()       {}
func (ns *NamespaceStatement) TokenLiteral() string { return ns.Token.Literal }
func (ns *NamespaceStatement) String() string {
	return "namespace " + ns.Name.Literal + ";"
}

// UseStatement represents 'use MyPackage\MyClass as Alias;'
type UseStatement struct {
	Token token.Token // 'use'
	Name  token.Token // Full name token
	Alias token.Token // Alias name token
}

func (us *UseStatement) statementNode()       {}
func (us *UseStatement) TokenLiteral() string { return us.Token.Literal }
func (us *UseStatement) String() string {
	if us.Alias.Literal != "" {
		return "use " + us.Name.Literal + " as " + us.Alias.Literal + ";"
	}
	return "use " + us.Name.Literal + ";"
}

type ArrayElement struct {
	Key   Expression // Can be nil
	Value Expression
}

type ArrayLiteral struct {
	Token    token.Token // '['
	Elements []*ArrayElement
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	var out bytes.Buffer
	out.WriteString("[")
	elems := []string{}
	for _, el := range al.Elements {
		if el.Key != nil {
			elems = append(elems, el.Key.String()+" => "+el.Value.String())
		} else {
			elems = append(elems, el.Value.String())
		}
	}
	out.WriteString(strings.Join(elems, ", "))
	out.WriteString("]")
	return out.String()
}

type IndexExpression struct {
	Token token.Token // '['
	Left  Expression  // The array/object being accessed
	Index Expression  // The index/key expression (nil if empty like $arr[])
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString("[")
	if ie.Index != nil {
		out.WriteString(ie.Index.String())
	}
	out.WriteString("])")
	return out.String()
}
