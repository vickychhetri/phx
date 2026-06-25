package compiler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"phx/internal/ast"
	"phx/internal/lexer"
	"phx/internal/parser"
	"phx/internal/token"
	"strings"
)

type Compiler struct {
	capturedVars []map[string]string
	inTryCatch   bool
	inFunction   bool
	dirStack     []string
}

func New() *Compiler {
	return &Compiler{
		capturedVars: make([]map[string]string, 0),
		inTryCatch:   false,
		inFunction:   false,
		dirStack:     make([]string, 0),
	}
}

func (c *Compiler) hasTryCatch(node ast.Node) bool {
	if node == nil {
		return false
	}
	switch n := node.(type) {
	case *ast.Program:
		for _, stmt := range n.Statements {
			if c.hasTryCatch(stmt) {
				return true
			}
		}
	case *ast.BlockStatement:
		for _, stmt := range n.Statements {
			if c.hasTryCatch(stmt) {
				return true
			}
		}
	case *ast.ExpressionStatement:
		return c.hasTryCatch(n.Expression)
	case *ast.IfStatement:
		return c.hasTryCatch(n.Condition) || c.hasTryCatch(n.Consequence) || c.hasTryCatch(n.Alternative)
	case *ast.ForStatement:
		return c.hasTryCatch(n.Init) || c.hasTryCatch(n.Condition) || c.hasTryCatch(n.Post) || c.hasTryCatch(n.Body)
	case *ast.WhileStatement:
		return c.hasTryCatch(n.Condition) || c.hasTryCatch(n.Body)
	case *ast.DoWhileStatement:
		return c.hasTryCatch(n.Condition) || c.hasTryCatch(n.Body)
	case *ast.TryCatchStatement:
		return true
	case *ast.IncludeStatement:
		strLit, ok := n.Expression.(*ast.StringLiteral)
		if !ok {
			return false
		}
		filename := strLit.Value
		resolvedPath := filename
		if !filepath.IsAbs(filename) {
			if len(c.dirStack) > 0 {
				p := filepath.Join(c.dirStack[len(c.dirStack)-1], filename)
				if _, err := os.Stat(p); err == nil {
					resolvedPath = p
				} else if len(c.dirStack) > 1 {
					p2 := filepath.Join(c.dirStack[0], filename)
					if _, err := os.Stat(p2); err == nil {
						resolvedPath = p2
					}
				}
			}
		}
		content, err := ioutil.ReadFile(resolvedPath)
		if err != nil {
			return false
		}
		l := lexer.New(string(content))
		p := parser.New(l)
		prog := p.ParseProgram()
		if len(p.Errors()) > 0 {
			return false
		}
		c.dirStack = append(c.dirStack, filepath.Dir(resolvedPath))
		res := c.hasTryCatch(prog)
		c.dirStack = c.dirStack[:len(c.dirStack)-1]
		return res
	}
	return false
}

func (c *Compiler) Compile(program *ast.Program, mainFilePath string) (string, error) {
	c.dirStack = []string{filepath.Dir(mainFilePath)}

	var buf bytes.Buffer
	buf.WriteString(goHeader)

	// Collect global variables
	globalVars := make(map[string]bool)
	for _, stmt := range program.Statements {
		c.collectVarsInBody(stmt, globalVars)
	}

	// Declare global variables in main
	buf.WriteString("func main() {\n")
	buf.WriteString(`	defer func() {
		if r := recover(); r != nil {
			if pe, ok := r.(PHXException); ok {
				var msg string
				if pe.Value.Type == 9 && pe.Value.Obj != nil {
					if res, err := pe.Value.Obj.Call("getMessage"); err == nil {
						msg = res.Str
					}
				}
				if msg == "" {
					msg = fmt.Sprintf("%v", pe.Value)
				}
				fmt.Fprintf(os.Stderr, "Fatal error: Uncaught exception: %s\n", msg)
				os.Exit(1)
			}
			panic(r)
		}
	}()
`)
	if c.hasTryCatch(program) {
		buf.WriteString("\tvar isReturn bool\n")
		buf.WriteString("\tvar returnVal Val\n")
		buf.WriteString("\t_, _ = isReturn, returnVal\n")
	}

	if len(globalVars) > 0 {
		var varNames []string
		for name := range globalVars {
			varNames = append(varNames, "v_"+name)
		}
		buf.WriteString(fmt.Sprintf("\tvar %s Val\n", strings.Join(varNames, ", ")))
		for _, name := range varNames {
			buf.WriteString(fmt.Sprintf("\t_ = %s\n", name))
		}
	}

	// Compile all statements in main
	for _, stmt := range program.Statements {
		buf.WriteString("\t" + c.compileNode(stmt) + "\n")
	}
	buf.WriteString("}\n")

	return buf.String(), nil
}

func (c *Compiler) isIntExpr(expr ast.Expression) (string, bool) {
	if expr == nil {
		return "", false
	}
	switch n := expr.(type) {
	case *ast.IntegerLiteral:
		return fmt.Sprintf("%d", n.Value), true
	case *ast.InfixExpression:
		leftStr, leftOk := c.isIntExpr(n.Left)
		rightStr, rightOk := c.isIntExpr(n.Right)
		if leftOk && rightOk {
			switch n.Operator {
			case "+", "-", "*", "/", "%":
				return fmt.Sprintf("(%s %s %s)", leftStr, n.Operator, rightStr), true
			}
		}
	}
	return "", false
}

func IsTruthyCode(code string) string {
	if strings.HasPrefix(code, "NewBool(") && strings.HasSuffix(code, ")") {
		return code[len("NewBool(") : len(code)-1]
	}
	return "IsTruthy(" + code + ")"
}

func (c *Compiler) compileNode(node ast.Node) string {
	if node == nil {
		return ""
	}

	switch n := node.(type) {
	case *ast.ExpressionStatement:
		return c.compileNode(n.Expression)

	case *ast.BlockStatement:
		var buf bytes.Buffer
		buf.WriteString("{\n")
		for _, stmt := range n.Statements {
			buf.WriteString("\t" + c.compileNode(stmt) + "\n")
		}
		buf.WriteString("}")
		return buf.String()

	case *ast.InlineHTMLStatement:
		return fmt.Sprintf("fmt.Print(%q)", n.Content)

	case *ast.EchoStatement:
		var parts []string
		for _, expr := range n.Expressions {
			parts = append(parts, c.compileNode(expr))
		}
		return fmt.Sprintf("echo(%s)", strings.Join(parts, ", "))

	case *ast.IfStatement:
		cond := IsTruthyCode(c.compileNode(n.Condition))
		consequence := c.compileNode(n.Consequence)
		alternative := ""
		if n.Alternative != nil {
			altCode := c.compileNode(n.Alternative)
			switch n.Alternative.(type) {
			case *ast.BlockStatement, *ast.IfStatement:
				alternative = " else " + altCode
			default:
				alternative = " else {\n\t" + altCode + "\n}"
			}
		}
		return fmt.Sprintf("if %s %s%s", cond, consequence, alternative)

	case *ast.WhileStatement:
		cond := IsTruthyCode(c.compileNode(n.Condition))
		body := c.compileNode(n.Body)
		return fmt.Sprintf("for %s %s", cond, body)

	case *ast.DoWhileStatement:
		body := c.compileNode(n.Body)
		cond := IsTruthyCode(c.compileNode(n.Condition))
		return fmt.Sprintf("for {\n\t%s\n\tif !(%s) {\n\t\tbreak\n\t}\n}", body, cond)

	case *ast.ForStatement:
		initCode := ""
		if n.Init != nil {
			initCode = c.compileNode(n.Init)
		}
		condCode := "true"
		if n.Condition != nil {
			condCode = IsTruthyCode(c.compileNode(n.Condition))
		}
		postCode := ""
		if n.Post != nil {
			postCode = c.compileNode(n.Post)
		}
		bodyCode := c.compileNode(n.Body)
		return fmt.Sprintf("{\n\t%s\n\tfor ; %s; %s %s\n}", initCode, condCode, postCode, bodyCode)

	case *ast.BreakStatement:
		return "break"

	case *ast.ContinueStatement:
		return "continue"

	case *ast.ReturnStatement:
		valCode := "Val{}"
		if n.ReturnValue != nil {
			valCode = c.compileNode(n.ReturnValue)
		}
		if c.inTryCatch {
			return fmt.Sprintf("{\n\t\treturnVal = %s\n\t\tisReturn = true\n\t\treturn\n\t}", valCode)
		}
		if c.inFunction {
			return "return " + valCode
		}
		return "return"

	case *ast.AssignExpression:
		op := n.Token.Literal
		leftCode := c.compileNode(n.Left)
		valCode := c.compileNode(n.Value)
		switch op {
		case "=":
			if ie, ok := n.Left.(*ast.IndexExpression); ok {
				if pe, ok2 := ie.Left.(*ast.PropertyExpression); ok2 {
					objCode := c.compileNode(pe.Object)
					propName := pe.Property.Value
					idxCode := "Val{}"
					if ie.Index != nil {
						idxCode = c.compileNode(ie.Index)
					}
					return fmt.Sprintf("SetPropertyIndex(%s, %q, %s, %s)", objCode, propName, idxCode, valCode)
				}
				arrayCode := c.compileNode(ie.Left)
				idxCode := "Val{}"
				if ie.Index != nil {
					idxCode = c.compileNode(ie.Index)
				}
				return fmt.Sprintf("SetIndex(&%s, %s, %s)", arrayCode, idxCode, valCode)
			}
			if pe, ok := n.Left.(*ast.PropertyExpression); ok {
				objCode := c.compileNode(pe.Object)
				propName := pe.Property.Value
				return fmt.Sprintf("SetProperty(%s, %q, %s)", objCode, propName, valCode)
			}
			return fmt.Sprintf("%s = %s", leftCode, valCode)
		case "+=", "-=", "*=":
			leftStr, leftOk := c.isIntExpr(n.Left)
			valStr, valOk := c.isIntExpr(n.Value)
			if leftOk && valOk {
				return fmt.Sprintf("%s %s %s", leftStr, op, valStr)
			}
			if pe, ok := n.Left.(*ast.PropertyExpression); ok {
				objCode := c.compileNode(pe.Object)
				propName := pe.Property.Value
				return fmt.Sprintf("AssignPropertyOp(%s, %q, %q, %s)", objCode, propName, op, valCode)
			}
			if op == "+=" {
				return fmt.Sprintf("AddAssign(&%s, %s)", leftCode, valCode)
			} else if op == "-=" {
				return fmt.Sprintf("SubAssign(&%s, %s)", leftCode, valCode)
			} else {
				return fmt.Sprintf("MulAssign(&%s, %s)", leftCode, valCode)
			}
		default:
			return fmt.Sprintf("%s = %s", leftCode, valCode)
		}

	case *ast.Variable:
		return c.getVarName(n.Value)

	case *ast.Identifier:
		return n.Value

	case *ast.IntegerLiteral:
		return fmt.Sprintf("NewInt(%d)", n.Value)

	case *ast.FloatLiteral:
		return fmt.Sprintf("NewFloat(%f)", n.Value)

	case *ast.StringLiteral:
		if n.Token.Type == token.T_DOUBLE_QUOTED_STRING {
			return c.compileStringLiteral(n.Value)
		}
		return fmt.Sprintf("NewStr(%q)", unescapeString(n.Value))

	case *ast.BooleanLiteral:
		return fmt.Sprintf("NewBool(%t)", n.Value)

	case *ast.NullLiteral:
		return "Val{}"

	case *ast.InfixExpression:
		leftStr, leftOk := c.isIntExpr(n.Left)
		rightStr, rightOk := c.isIntExpr(n.Right)
		if leftOk && rightOk {
			switch n.Operator {
			case "+", "-", "*", "/", "%":
				return fmt.Sprintf("NewInt(%s %s %s)", leftStr, n.Operator, rightStr)
			case "<", ">", "<=", ">=":
				return fmt.Sprintf("NewBool(%s %s %s)", leftStr, n.Operator, rightStr)
			case "==", "===":
				return fmt.Sprintf("NewBool(%s == %s)", leftStr, rightStr)
			case "!=", "!==":
				return fmt.Sprintf("NewBool(%s != %s)", leftStr, rightStr)
			}
		}

		left := c.compileNode(n.Left)
		right := c.compileNode(n.Right)
		switch n.Operator {
		case "+": return fmt.Sprintf("Add(%s, %s)", left, right)
		case "-": return fmt.Sprintf("Sub(%s, %s)", left, right)
		case "*": return fmt.Sprintf("Mul(%s, %s)", left, right)
		case "/": return fmt.Sprintf("Div(%s, %s)", left, right)
		case "%": return fmt.Sprintf("Mod(%s, %s)", left, right)
		case "<": return fmt.Sprintf("Lt(%s, %s)", left, right)
		case ">": return fmt.Sprintf("Gt(%s, %s)", left, right)
		case "<=": return fmt.Sprintf("Le(%s, %s)", left, right)
		case ">=": return fmt.Sprintf("Ge(%s, %s)", left, right)
		case "==", "===": return fmt.Sprintf("Eq(%s, %s)", left, right)
		case "!=", "!==": return fmt.Sprintf("Ne(%s, %s)", left, right)
		case ".": return fmt.Sprintf("Concat(%s, %s)", left, right)
		default:
			return fmt.Sprintf("Add(%s, %s)", left, right)
		}

	case *ast.PrefixExpression:
		rightStr, rightOk := c.isIntExpr(n.Right)
		if rightOk {
			switch n.Operator {
			case "++": return rightStr + "++"
			case "--": return rightStr + "--"
			}
		}
		if pe, ok := n.Right.(*ast.PropertyExpression); ok {
			objCode := c.compileNode(pe.Object)
			propName := pe.Property.Value
			if n.Operator == "++" {
				return fmt.Sprintf("PreIncProperty(%s, %q)", objCode, propName)
			} else {
				return fmt.Sprintf("PreDecProperty(%s, %q)", objCode, propName)
			}
		}

		right := c.compileNode(n.Right)
		switch n.Operator {
		case "++": return fmt.Sprintf("PrefixInc(&%s)", right)
		case "--": return fmt.Sprintf("PrefixDec(&%s)", right)
		case "!": return fmt.Sprintf("Not(%s)", right)
		case "-": return fmt.Sprintf("Neg(%s)", right)
		default:
			return right
		}

	case *ast.PostExpression:
		leftStr, leftOk := c.isIntExpr(n.Left)
		if leftOk {
			switch n.Operator {
			case "++": return leftStr + "++"
			case "--": return leftStr + "--"
			}
		}
		if pe, ok := n.Left.(*ast.PropertyExpression); ok {
			objCode := c.compileNode(pe.Object)
			propName := pe.Property.Value
			if n.Operator == "++" {
				return fmt.Sprintf("PostIncProperty(%s, %q)", objCode, propName)
			} else {
				return fmt.Sprintf("PostDecProperty(%s, %q)", objCode, propName)
			}
		}

		left := c.compileNode(n.Left)
		switch n.Operator {
		case "++": return fmt.Sprintf("PostfixInc(&%s)", left)
		case "--": return fmt.Sprintf("PostfixDec(&%s)", left)
		default:
			return left
		}

	case *ast.CallExpression:
		var fn string
		if ident, ok := n.Function.(*ast.Identifier); ok {
			if isBuiltin(ident.Value) {
				fn = ident.Value
			} else {
				fn = "v_" + ident.Value
			}
		} else {
			fn = c.compileNode(n.Function)
		}
		var args []string
		for _, arg := range n.Arguments {
			args = append(args, c.compileNode(arg))
		}
		return fmt.Sprintf("Call(%s, %s)", fn, strings.Join(args, ", "))

	case *ast.ArrayLiteral:
		var parts []string
		hasKeys := false
		for _, el := range n.Elements {
			if el.Key != nil {
				hasKeys = true
			}
			parts = append(parts, c.compileNode(el.Value))
		}
		if hasKeys {
			var assocParts []string
			for _, el := range n.Elements {
				keyStr := "Val{}"
				if el.Key != nil {
					keyStr = c.compileNode(el.Key)
				}
				assocParts = append(assocParts, keyStr, c.compileNode(el.Value))
			}
			return fmt.Sprintf("NewAssociativeArray(%s)", strings.Join(assocParts, ", "))
		}
		return fmt.Sprintf("NewArray(%s)", strings.Join(parts, ", "))

	case *ast.IndexExpression:
		arr := c.compileNode(n.Left)
		idx := "Val{}"
		if n.Index != nil {
			idx = c.compileNode(n.Index)
		}
		return fmt.Sprintf("GetIndex(%s, %s)", arr, idx)

	case *ast.FunctionExpression:
		var buf bytes.Buffer

		// Copy-capture variables specified in the use(...) clause
		hasCapture := len(n.UseVars) > 0
		if hasCapture {
			buf.WriteString("func() Val {\n")
			capturedMap := make(map[string]string)
			for _, useVar := range n.UseVars {
				cleanName := strings.TrimPrefix(useVar.Value, "$")
				capturedName := "captured_" + cleanName
				capturedMap[cleanName] = capturedName
				// Generate: captured_x := v_x
				buf.WriteString(fmt.Sprintf("\t\t%s := v_%s\n", capturedName, cleanName))
			}
			c.capturedVars = append(c.capturedVars, capturedMap)
			buf.WriteString("\t\treturn ")
		}

		buf.WriteString("Val{Type: 8, Func: func(args ...Val) Val {\n")
		// Declare parameters
		for i, param := range n.Parameters {
			varName := "v_" + strings.TrimPrefix(param.Var.Value, "$")
			defaultVal := "Val{}"
			if param.DefaultValue != nil {
				defaultVal = c.compileNode(param.DefaultValue)
			}
			buf.WriteString(fmt.Sprintf("\t\tvar %s Val\n", varName))
			buf.WriteString(fmt.Sprintf("\t\tif len(args) > %d {\n", i))
			buf.WriteString(fmt.Sprintf("\t\t\t%s = args[%d]\n", varName, i))
			buf.WriteString("\t\t} else {\n")
			buf.WriteString(fmt.Sprintf("\t\t\t%s = %s\n", varName, defaultVal))
			buf.WriteString("\t\t}\n")
		}

		if len(n.Parameters) > 0 {
			for _, param := range n.Parameters {
				buf.WriteString(fmt.Sprintf("\t\t_ = v_%s\n", strings.TrimPrefix(param.Var.Value, "$")))
			}
		}

		// Collect local variables
		localVars := make(map[string]bool)
		c.collectVarsInBody(n.Body, localVars)
		for _, param := range n.Parameters {
			delete(localVars, strings.TrimPrefix(param.Var.Value, "$"))
		}
		if len(localVars) > 0 {
			var varNames []string
			for name := range localVars {
				varNames = append(varNames, "v_"+name)
			}
			buf.WriteString(fmt.Sprintf("\t\tvar %s Val\n", strings.Join(varNames, ", ")))
			for _, name := range varNames {
				buf.WriteString(fmt.Sprintf("\t\t_ = %s\n", name))
			}
		}

		if c.hasTryCatch(n.Body) {
			buf.WriteString("\t\tvar isReturn bool\n")
			buf.WriteString("\t\tvar returnVal Val\n")
			buf.WriteString("\t\t_, _ = isReturn, returnVal\n")
		}

		// Compile statements
		oldInFunction := c.inFunction
		c.inFunction = true
		for _, stmt := range n.Body.Statements {
			buf.WriteString("\t\t" + c.compileNode(stmt) + "\n")
		}
		c.inFunction = oldInFunction

		buf.WriteString("\t\treturn Val{}\n")
		buf.WriteString("\t}}")

		if hasCapture {
			buf.WriteString("\n\t}()")
			c.capturedVars = c.capturedVars[:len(c.capturedVars)-1]
		}

		return buf.String()

	case *ast.ClassStatement:
		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("classes[%q] = map[string]Val{\n", n.Name.Value))
		for _, m := range n.Methods {
			methodName := m.Name.Value
			buf.WriteString(fmt.Sprintf("\t%q: Val{Type: 8, Func: func(args ...Val) Val {\n", methodName))
			buf.WriteString("\t\tv_this := args[0]\n")
			buf.WriteString("\t\t_ = v_this\n")
			for i, param := range m.Parameters {
				varName := "v_" + strings.TrimPrefix(param.Var.Value, "$")
				defaultVal := "Val{}"
				if param.DefaultValue != nil {
					defaultVal = c.compileNode(param.DefaultValue)
				}
				buf.WriteString(fmt.Sprintf("\t\tvar %s Val\n", varName))
				buf.WriteString(fmt.Sprintf("\t\tif len(args) > %d {\n", i+1))
				buf.WriteString(fmt.Sprintf("\t\t\t%s = args[%d]\n", varName, i+1))
				buf.WriteString("\t\t} else {\n")
				buf.WriteString(fmt.Sprintf("\t\t\t%s = %s\n", varName, defaultVal))
				buf.WriteString("\t\t}\n")
			}
			if len(m.Parameters) > 0 {
				for _, param := range m.Parameters {
					buf.WriteString(fmt.Sprintf("\t\t_ = v_%s\n", strings.TrimPrefix(param.Var.Value, "$")))
				}
			}
			localVars := make(map[string]bool)
			c.collectVarsInBody(m.Body, localVars)
			for _, param := range m.Parameters {
				delete(localVars, strings.TrimPrefix(param.Var.Value, "$"))
			}
			delete(localVars, "this")
			if len(localVars) > 0 {
				var varNames []string
				for name := range localVars {
					varNames = append(varNames, "v_"+name)
				}
				buf.WriteString(fmt.Sprintf("\t\tvar %s Val\n", strings.Join(varNames, ", ")))
				for _, name := range varNames {
					buf.WriteString(fmt.Sprintf("\t\t_ = %s\n", name))
				}
			}
			if c.hasTryCatch(m.Body) {
				buf.WriteString("\t\tvar isReturn bool\n")
				buf.WriteString("\t\tvar returnVal Val\n")
				buf.WriteString("\t\t_, _ = isReturn, returnVal\n")
			}
			oldInFunction := c.inFunction
			c.inFunction = true
			for _, stmt := range m.Body.Statements {
				buf.WriteString("\t\t" + c.compileNode(stmt) + "\n")
			}
			c.inFunction = oldInFunction
			buf.WriteString("\t\treturn Val{}\n")
			buf.WriteString("\t}},\n")
		}
		buf.WriteString("}")
		return buf.String()

	case *ast.NewExpression:
		className := n.Class.Value
		var args []string
		for _, arg := range n.Arguments {
			args = append(args, c.compileNode(arg))
		}
		return fmt.Sprintf("NewObject(%q, %s)", className, strings.Join(args, ", "))

	case *ast.MethodCallExpression:
		objCode := c.compileNode(n.Object)
		methodName := n.Method.Value
		var args []string
		for _, arg := range n.Arguments {
			args = append(args, c.compileNode(arg))
		}
		return fmt.Sprintf("CallMethod(%s, %q, %s)", objCode, methodName, strings.Join(args, ", "))

	case *ast.PropertyExpression:
		objCode := c.compileNode(n.Object)
		propName := n.Property.Value
		return fmt.Sprintf("GetProperty(%s, %q)", objCode, propName)

	case *ast.ThrowStatement:
		valCode := c.compileNode(n.Expression)
		return fmt.Sprintf("panic(PHXException{Value: %s})", valCode)

	case *ast.TryCatchStatement:
		var buf bytes.Buffer
		buf.WriteString("func() {\n")
		buf.WriteString("\tdefer func() {\n")
		buf.WriteString("\t\tif r := recover(); r != nil {\n")
		buf.WriteString("\t\t\tif pe, ok := r.(PHXException); ok {\n")
		catchVarName := "v_" + strings.TrimPrefix(n.CatchVar, "$")
		buf.WriteString(fmt.Sprintf("\t\t\t\t%s = pe.Value\n", catchVarName))
		buf.WriteString("\t\t\t\t_ = " + catchVarName + "\n")
		
		oldInTryCatch := c.inTryCatch
		c.inTryCatch = true
		for _, stmt := range n.CatchBlock.Statements {
			buf.WriteString("\t\t\t\t" + c.compileNode(stmt) + "\n")
		}
		c.inTryCatch = oldInTryCatch

		buf.WriteString("\t\t\t} else {\n")
		buf.WriteString("\t\t\t\tpanic(r)\n")
		buf.WriteString("\t\t\t}\n")
		buf.WriteString("\t\t}\n")
		buf.WriteString("\t}()\n")
		
		oldInTryCatch = c.inTryCatch
		c.inTryCatch = true
		for _, stmt := range n.TryBlock.Statements {
			buf.WriteString("\t" + c.compileNode(stmt) + "\n")
		}
		c.inTryCatch = oldInTryCatch

		buf.WriteString("}()\n")
		if c.inFunction {
			buf.WriteString("\tif isReturn {\n\t\treturn returnVal\n\t}")
		} else {
			buf.WriteString("\tif isReturn {\n\t\treturn\n\t}")
		}
		return buf.String()

	case *ast.FunctionStatement:
		varName := "v_" + strings.TrimPrefix(n.Name.Value, "$")
		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("%s = Val{Type: 8, Func: func(args ...Val) Val {\n", varName))
		for i, param := range n.Parameters {
			paramVarName := "v_" + strings.TrimPrefix(param.Var.Value, "$")
			defaultVal := "Val{}"
			if param.DefaultValue != nil {
				defaultVal = c.compileNode(param.DefaultValue)
			}
			buf.WriteString(fmt.Sprintf("\t\tvar %s Val\n", paramVarName))
			buf.WriteString(fmt.Sprintf("\t\tif len(args) > %d {\n", i))
			buf.WriteString(fmt.Sprintf("\t\t\t%s = args[%d]\n", paramVarName, i))
			buf.WriteString("\t\t} else {\n")
			buf.WriteString(fmt.Sprintf("\t\t\t%s = %s\n", paramVarName, defaultVal))
			buf.WriteString("\t\t}\n")
		}
		if len(n.Parameters) > 0 {
			for _, param := range n.Parameters {
				buf.WriteString(fmt.Sprintf("\t\t_ = v_%s\n", strings.TrimPrefix(param.Var.Value, "$")))
			}
		}
		localVars := make(map[string]bool)
		c.collectVarsInBody(n.Body, localVars)
		for _, param := range n.Parameters {
			delete(localVars, strings.TrimPrefix(param.Var.Value, "$"))
		}
		if len(localVars) > 0 {
			var varNames []string
			for name := range localVars {
				varNames = append(varNames, "v_"+name)
			}
			buf.WriteString(fmt.Sprintf("\t\tvar %s Val\n", strings.Join(varNames, ", ")))
			for _, name := range varNames {
				buf.WriteString(fmt.Sprintf("\t\t_ = %s\n", name))
			}
		}
		if c.hasTryCatch(n.Body) {
			buf.WriteString("\t\tvar isReturn bool\n")
			buf.WriteString("\t\tvar returnVal Val\n")
			buf.WriteString("\t\t_, _ = isReturn, returnVal\n")
		}
		oldInFunction := c.inFunction
		c.inFunction = true
		for _, stmt := range n.Body.Statements {
			buf.WriteString("\t\t" + c.compileNode(stmt) + "\n")
		}
		c.inFunction = oldInFunction
		buf.WriteString("\t\treturn Val{}\n")
		buf.WriteString("\t}}")
		return buf.String()

	case *ast.SwitchStatement:
		var buf bytes.Buffer
		buf.WriteString("{\n")
		exprCode := c.compileNode(n.Expr)
		buf.WriteString(fmt.Sprintf("\tval_switch := %s\n", exprCode))
		buf.WriteString("\tfor {\n")
		
		for i, cs := range n.Cases {
			caseValCode := c.compileNode(cs.Value)
			cond := fmt.Sprintf("IsTruthy(Eq(val_switch, %s))", caseValCode)
			if i == 0 {
				buf.WriteString(fmt.Sprintf("\t\tif %s {\n", cond))
			} else {
				buf.WriteString(fmt.Sprintf("\t\t} else if %s {\n", cond))
			}
			for _, stmt := range cs.Body.Statements {
				buf.WriteString("\t\t\t" + c.compileNode(stmt) + "\n")
			}
		}
		if n.Default != nil {
			if len(n.Cases) > 0 {
				buf.WriteString("\t\t} else {\n")
			} else {
				buf.WriteString("\t\t{\n")
			}
			for _, stmt := range n.Default.Statements {
				buf.WriteString("\t\t\t" + c.compileNode(stmt) + "\n")
			}
		}
		if len(n.Cases) > 0 || n.Default != nil {
			buf.WriteString("\t\t}\n")
		}
		buf.WriteString("\t\tbreak\n")
		buf.WriteString("\t}\n")
		buf.WriteString("}")
		return buf.String()

	case *ast.TernaryExpression:
		cond := IsTruthyCode(c.compileNode(n.Condition))
		conseq := c.compileNode(n.Consequence)
		alt := c.compileNode(n.Alternative)
		return fmt.Sprintf("func() Val {\n\t\tif %s {\n\t\t\treturn %s\n\t\t}\n\t\treturn %s\n\t}()", cond, conseq, alt)

	case *ast.IncludeStatement:
		strLit, ok := n.Expression.(*ast.StringLiteral)
		if !ok {
			return "/* include with non-string literal expression is unsupported in compiled mode */"
		}
		filename := strLit.Value
		resolvedPath := filename
		if !filepath.IsAbs(filename) {
			if len(c.dirStack) > 0 {
				p := filepath.Join(c.dirStack[len(c.dirStack)-1], filename)
				if _, err := os.Stat(p); err == nil {
					resolvedPath = p
				} else if len(c.dirStack) > 1 {
					p2 := filepath.Join(c.dirStack[0], filename)
					if _, err := os.Stat(p2); err == nil {
						resolvedPath = p2
					}
				}
			}
		}
		content, err := ioutil.ReadFile(resolvedPath)
		if err != nil {
			return fmt.Sprintf("panic(\"failed to read include file: %s\")", filename)
		}
		l := lexer.New(string(content))
		p := parser.New(l)
		prog := p.ParseProgram()
		if len(p.Errors()) > 0 {
			return fmt.Sprintf("panic(\"parser errors in include file: %s\")", filename)
		}
		c.dirStack = append(c.dirStack, filepath.Dir(resolvedPath))
		var buf bytes.Buffer
		for _, stmt := range prog.Statements {
			buf.WriteString(c.compileNode(stmt) + "\n")
		}
		c.dirStack = c.dirStack[:len(c.dirStack)-1]
		return buf.String()

	default:
		return fmt.Sprintf("/* unsupported: %T */", node)
	}
}

func (c *Compiler) collectVarsInBody(node ast.Node, localVars map[string]bool) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *ast.Program:
		for _, stmt := range n.Statements {
			c.collectVarsInBody(stmt, localVars)
		}
	case *ast.BlockStatement:
		for _, stmt := range n.Statements {
			c.collectVarsInBody(stmt, localVars)
		}
	case *ast.ExpressionStatement:
		c.collectVarsInBody(n.Expression, localVars)
	case *ast.AssignExpression:
		if v, ok := n.Left.(*ast.Variable); ok {
			localVars[strings.TrimPrefix(v.Value, "$")] = true
		}
		c.collectVarsInBody(n.Value, localVars)
	case *ast.FunctionStatement:
		localVars[strings.TrimPrefix(n.Name.Value, "$")] = true
	case *ast.ForStatement:
		c.collectVarsInBody(n.Init, localVars)
		c.collectVarsInBody(n.Condition, localVars)
		c.collectVarsInBody(n.Post, localVars)
		c.collectVarsInBody(n.Body, localVars)
	case *ast.WhileStatement:
		c.collectVarsInBody(n.Condition, localVars)
		c.collectVarsInBody(n.Body, localVars)
	case *ast.DoWhileStatement:
		c.collectVarsInBody(n.Condition, localVars)
		c.collectVarsInBody(n.Body, localVars)
	case *ast.IfStatement:
		c.collectVarsInBody(n.Condition, localVars)
		c.collectVarsInBody(n.Consequence, localVars)
		c.collectVarsInBody(n.Alternative, localVars)
	case *ast.TryCatchStatement:
		localVars[strings.TrimPrefix(n.CatchVar, "$")] = true
		c.collectVarsInBody(n.TryBlock, localVars)
		c.collectVarsInBody(n.CatchBlock, localVars)
	case *ast.ThrowStatement:
		c.collectVarsInBody(n.Expression, localVars)
	case *ast.IncludeStatement:
		strLit, ok := n.Expression.(*ast.StringLiteral)
		if !ok {
			return
		}
		filename := strLit.Value
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			return
		}
		l := lexer.New(string(content))
		p := parser.New(l)
		prog := p.ParseProgram()
		if len(p.Errors()) > 0 {
			return
		}
		for _, stmt := range prog.Statements {
			c.collectVarsInBody(stmt, localVars)
		}
	}
}

func (c *Compiler) getVarName(rawName string) string {
	cleanName := strings.TrimPrefix(rawName, "$")
	for i := len(c.capturedVars) - 1; i >= 0; i-- {
		if capName, ok := c.capturedVars[i][cleanName]; ok {
			return capName
		}
	}
	return "v_" + cleanName
}

func (c *Compiler) compileStringLiteral(s string) string {
	var parts []string
	i := 0
	n := len(s)
	curr := ""
	for i < n {
		if i+1 < n && s[i] == '{' && s[i+1] == '$' {
			if curr != "" {
				parts = append(parts, fmt.Sprintf("NewStr(%q)", unescapeString(curr)))
				curr = ""
			}
			start := i + 1 // points to '$'
			end := start
			braces := 1
			for end < n {
				if s[end] == '{' {
					braces++
				} else if s[end] == '}' {
					braces--
					if braces == 0 {
						break
					}
				}
				end++
			}
			if end < n {
				varName := s[start+1 : end]
				parts = append(parts, fmt.Sprintf("ToString(%s)", c.getVarName(varName)))
				i = end + 1
				continue
			}
		} else if s[i] == '$' {
			nextIdx := i + 1
			for nextIdx < n && (isAlphaNumeric(s[nextIdx]) || s[nextIdx] == '_') {
				nextIdx++
			}
			varName := s[i+1 : nextIdx]
			if varName == "" {
				curr += "$"
				i++
				continue
			}
			if curr != "" {
				parts = append(parts, fmt.Sprintf("NewStr(%q)", unescapeString(curr)))
				curr = ""
			}
			parts = append(parts, fmt.Sprintf("ToString(%s)", c.getVarName(varName)))
			i = nextIdx
			continue
		}
		curr += string(s[i])
		i++
	}
	if curr != "" {
		parts = append(parts, fmt.Sprintf("NewStr(%q)", unescapeString(curr)))
	}
	if len(parts) == 0 {
		return `NewStr("")`
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return fmt.Sprintf("Concat(%s)", strings.Join(parts, ", "))
}

func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func unescapeString(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result = append(result, '\n')
			case 't':
				result = append(result, '\t')
			case 'r':
				result = append(result, '\r')
			case '\\':
				result = append(result, '\\')
			case '"':
				result = append(result, '"')
			case '\'':
				result = append(result, '\'')
			case '$':
				result = append(result, '$')
			default:
				result = append(result, s[i])
				result = append(result, s[i+1])
			}
			i++
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}

func isBuiltin(name string) bool {
	switch name {
	case "count", "microtime", "channel", "send", "receive", "spawn", "intdiv",
		"sleep", "print_r", "strtoupper", "strtolower", "str_repeat", "trim",
		"str_replace", "number_format", "strlen", "strpos", "substr", "readline",
		"intval", "floatval", "strval", "abs", "min", "max", "rand":
		return true
	}
	return false
}

const goHeader = `package main

import (
	"fmt"
	"time"
	"strings"
	"bytes"
	"database/sql"
	"os"
	"io"
	"bufio"
	"net/http"
	"encoding/json"
	"regexp"
	"sync"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type Val struct {
	Type  int // 0: Nil, 1: Int, 2: Float, 3: Bool, 4: String, 5: Array, 6: Chan, 7: Thread, 8: Func, 9: Object
	Int   int64
	Float float64
	Bool  bool
	Str   string
	Array []Val
	Chan  chan Val
	Func  func(args ...Val) Val
	Obj   Object
}

type PHXException struct {
	Value Val
}

type Object interface {
	Get(prop string) Val
	Set(prop string, val Val)
	Call(method string, args ...Val) (Val, error)
}

type PHXObject struct {
	ClassName string
	Fields    map[string]Val
	Methods   map[string]Val
}

func (o *PHXObject) Get(prop string) Val {
	if o.Fields == nil {
		return Val{}
	}
	return o.Fields[prop]
}

func (o *PHXObject) Set(prop string, val Val) {
	if o.Fields == nil {
		o.Fields = make(map[string]Val)
	}
	o.Fields[prop] = val
}

func (o *PHXObject) Call(method string, args ...Val) (Val, error) {
	m, ok := o.Methods[method]
	if !ok {
		return Val{}, fmt.Errorf("undefined method: %s on class %s", method, o.ClassName)
	}
	thisVal := Val{Type: 9, Obj: o}
	methodArgs := append([]Val{thisVal}, args...)
	return m.Func(methodArgs...), nil
}

type PHXExceptionObject struct {
	Message string
}

func (e *PHXExceptionObject) Get(prop string) Val { return Val{} }
func (e *PHXExceptionObject) Set(prop string, val Val) {}
func (e *PHXExceptionObject) Call(method string, args ...Val) (Val, error) {
	if method == "getMessage" {
		return Val{Type: 4, Str: e.Message}, nil
	}
	return Val{}, fmt.Errorf("undefined method: %s on Exception", method)
}

type FileObject struct {
	file *os.File
}

func (f *FileObject) Get(prop string) Val { return Val{} }
func (f *FileObject) Set(prop string, val Val) {}
func (f *FileObject) Call(method string, args ...Val) (Val, error) {
	switch method {
	case "open":
		if len(args) < 2 {
			return Val{}, fmt.Errorf("File::open expects 2 arguments: path and mode")
		}
		path := ToString(args[0]).Str
		mode := ToString(args[1]).Str

		var flag int
		if strings.Contains(mode, "w") {
			flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		} else if strings.Contains(mode, "a") {
			flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
		} else {
			flag = os.O_RDONLY
		}

		file, err := os.OpenFile(path, flag, 0666)
		if err != nil {
			return Val{}, err
		}
		f.file = file
		return Val{Type: 3, Bool: true}, nil

	case "read":
		if f.file == nil {
			return Val{}, fmt.Errorf("file is not open")
		}
		if len(args) < 1 {
			return Val{}, fmt.Errorf("File::read expects 1 argument: length")
		}
		length := args[0].Int
		buf := make([]byte, length)
		nBytes, err := f.file.Read(buf)
		if err != nil && err != io.EOF {
			return Val{}, err
		}
		return NewStr(string(buf[:nBytes])), nil

	case "readLine":
		if f.file == nil {
			return Val{}, fmt.Errorf("file is not open")
		}
		reader := bufio.NewReader(f.file)
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return Val{}, err
		}
		if line == "" && err == io.EOF {
			return Val{Type: 3, Bool: false}, nil
		}
		return NewStr(line), nil

	case "write":
		if f.file == nil {
			return Val{}, fmt.Errorf("file is not open")
		}
		if len(args) < 1 {
			return Val{}, fmt.Errorf("File::write expects 1 argument: content")
		}
		content := ToString(args[0]).Str
		nBytes, err := f.file.WriteString(content)
		if err != nil {
			return Val{}, err
		}
		return NewInt(int64(nBytes)), nil

	case "close":
		if f.file != nil {
			f.file.Close()
			f.file = nil
		}
		return Val{Type: 3, Bool: true}, nil

	case "exists":
		if len(args) < 1 {
			return Val{}, fmt.Errorf("File::exists expects 1 argument: path")
		}
		path := ToString(args[0]).Str
		_, err := os.Stat(path)
		if err == nil {
			return Val{Type: 3, Bool: true}, nil
		}
		return Val{Type: 3, Bool: false}, nil

	case "delete":
		if len(args) < 1 {
			return Val{}, fmt.Errorf("File::delete expects 1 argument: path")
		}
		path := ToString(args[0]).Str
		err := os.Remove(path)
		if err != nil {
			return Val{Type: 3, Bool: false}, nil
		}
		return Val{Type: 3, Bool: true}, nil
	}
	return Val{}, fmt.Errorf("undefined method: %s on FileObject", method)
}

type MySQLObject struct {
	dbConn    *sql.DB
	connected bool
}

func (m *MySQLObject) Get(prop string) Val { return Val{} }
func (m *MySQLObject) Set(prop string, val Val) {}
func (m *MySQLObject) Call(method string, args ...Val) (Val, error) {
	switch method {
	case "connect":
		if len(args) < 4 {
			return Val{}, fmt.Errorf("MySQL::connect expects 4 arguments")
		}
		host := ToString(args[0]).Str
		user := ToString(args[1]).Str
		password := ToString(args[2]).Str
		db := ToString(args[3]).Str
		
		dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", user, password, host, db)
		conn, err := sql.Open("mysql", dsn)
		if err == nil {
			err = conn.Ping()
		}
		if err != nil {
			dsnNoDB := fmt.Sprintf("%s:%s@tcp(%s)/?parseTime=true", user, password, host)
			connNoDB, err2 := sql.Open("mysql", dsnNoDB)
			if err2 == nil {
				if err2 = connNoDB.Ping(); err2 == nil {
					_, _ = connNoDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", db))
					connNoDB.Close()
					conn, err = sql.Open("mysql", dsn)
					if err == nil {
						err = conn.Ping()
					}
				}
			}
		}
		if err != nil {
			return Val{}, err
		}
		m.dbConn = conn
		m.connected = true
		return Val{Type: 3, Bool: true}, nil

	case "exec":
		if !m.connected {
			return Val{}, fmt.Errorf("MySQL Error: Not connected")
		}
		query := ToString(args[0]).Str
		_, err := m.dbConn.Exec(query)
		if err != nil {
			return Val{}, err
		}
		return Val{Type: 3, Bool: true}, nil

	case "query":
		if !m.connected {
			return Val{}, fmt.Errorf("MySQL Error: Not connected")
		}
		query := ToString(args[0]).Str
		rows, err := m.dbConn.Query(query)
		if err != nil {
			return Val{}, err
		}
		defer rows.Close()
		return scanSQLRows(rows)

	case "close":
		if m.dbConn != nil {
			m.dbConn.Close()
			m.dbConn = nil
		}
		m.connected = false
		return Val{Type: 3, Bool: true}, nil
	}
	return Val{}, fmt.Errorf("undefined method: %s on MySQLObject", method)
}

type PostgresObject struct {
	dbConn    *sql.DB
	connected bool
}

func (p *PostgresObject) Get(prop string) Val { return Val{} }
func (p *PostgresObject) Set(prop string, val Val) {}
func (p *PostgresObject) Call(method string, args ...Val) (Val, error) {
	switch method {
	case "connect":
		if len(args) < 4 {
			return Val{}, fmt.Errorf("PostgreSQL::connect expects at least 4 arguments")
		}
		host := ToString(args[0]).Str
		user := ToString(args[1]).Str
		password := ToString(args[2]).Str
		db := ToString(args[3]).Str
		var port int64 = 5432
		if len(args) > 4 && args[4].Type == 1 {
			port = args[4].Int
		}
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, db)
		conn, err := sql.Open("postgres", dsn)
		if err == nil {
			err = conn.Ping()
		}
		if err != nil {
			dsnNoDB := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable", host, port, user, password)
			connNoDB, err2 := sql.Open("postgres", dsnNoDB)
			if err2 == nil {
				if err2 = connNoDB.Ping(); err2 == nil {
					var exists bool
					_ = connNoDB.QueryRow("SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = $1)", db).Scan(&exists)
					if !exists {
						_, _ = connNoDB.Exec(fmt.Sprintf("CREATE DATABASE %s", db))
					}
					connNoDB.Close()
					conn, err = sql.Open("postgres", dsn)
					if err == nil {
						err = conn.Ping()
					}
				}
			}
		}
		if err != nil {
			return Val{}, err
		}
		p.dbConn = conn
		p.connected = true
		return Val{Type: 3, Bool: true}, nil

	case "exec":
		if !p.connected {
			return Val{}, fmt.Errorf("PostgreSQL Error: Not connected")
		}
		query := ToString(args[0]).Str
		_, err := p.dbConn.Exec(query)
		if err != nil {
			return Val{}, err
		}
		return Val{Type: 3, Bool: true}, nil

	case "query":
		if !p.connected {
			return Val{}, fmt.Errorf("PostgreSQL Error: Not connected")
		}
		query := ToString(args[0]).Str
		rows, err := p.dbConn.Query(query)
		if err != nil {
			return Val{}, err
		}
		defer rows.Close()
		return scanSQLRows(rows)

	case "close":
		if p.dbConn != nil {
			p.dbConn.Close()
			p.dbConn = nil
		}
		p.connected = false
		return Val{Type: 3, Bool: true}, nil
	}
	return Val{}, fmt.Errorf("undefined method: %s on PostgreSQLConnection", method)
}

type MongoObject struct {
	uri         string
	connected   bool
	dbName      string
	collName    string
	collections map[string][]Val
}

func (m *MongoObject) Get(prop string) Val { return Val{} }
func (m *MongoObject) Set(prop string, val Val) {}
func (m *MongoObject) Call(method string, args ...Val) (Val, error) {
	switch method {
	case "connect":
		if len(args) < 1 {
			return Val{}, fmt.Errorf("MongoDB::connect expects 1 argument")
		}
		m.uri = ToString(args[0]).Str
		m.connected = true
		fmt.Printf("[MongoDB] Connected to %s\n", m.uri)
		return Val{Type: 3, Bool: true}, nil

	case "selectDatabase":
		if !m.connected {
			return Val{}, fmt.Errorf("MongoDB Error: Not connected")
		}
		if len(args) < 1 {
			return Val{}, fmt.Errorf("MongoDB::selectDatabase expects 1 argument")
		}
		m.dbName = ToString(args[0]).Str
		return Val{Type: 3, Bool: true}, nil

	case "selectCollection":
		if !m.connected {
			return Val{}, fmt.Errorf("MongoDB Error: Not connected")
		}
		if len(args) < 1 {
			return Val{}, fmt.Errorf("MongoDB::selectCollection expects 1 argument")
		}
		m.collName = ToString(args[0]).Str
		return Val{Type: 3, Bool: true}, nil

	case "insertOne":
		if !m.connected {
			return Val{}, fmt.Errorf("MongoDB Error: Not connected")
		}
		if len(args) < 1 {
			return Val{}, fmt.Errorf("MongoDB::insertOne expects 1 argument")
		}
		doc := args[0]
		if m.collections == nil {
			m.collections = make(map[string][]Val)
		}
		m.collections[m.collName] = append(m.collections[m.collName], doc)
		return Val{Type: 3, Bool: true}, nil

	case "find":
		if !m.connected {
			return Val{}, fmt.Errorf("MongoDB Error: Not connected")
		}
		filter := Val{}
		if len(args) > 0 {
			filter = args[0]
		}
		docs, ok := m.collections[m.collName]
		if !ok {
			return NewArray(), nil
		}
		var results []Val
		for _, doc := range docs {
			matches := true
			if filter.Type == 5 && len(filter.Array) > 0 {
				for i := 0; i < len(filter.Array); i += 2 {
					filterKey := filter.Array[i].Str
					filterVal := ToString(filter.Array[i+1]).Str
					
					fieldMatched := false
					if doc.Type == 5 {
						for j := 0; j < len(doc.Array); j += 2 {
							docKey := doc.Array[j].Str
							if docKey == filterKey {
								docVal := ToString(doc.Array[j+1]).Str
								if docVal == filterVal {
									fieldMatched = true
									break
								}
							}
						}
					}
					if !fieldMatched {
						matches = false
						break
					}
				}
			}
			if matches {
				results = append(results, doc)
			}
		}
		return NewArray(results...), nil

	case "close":
		m.connected = false
		fmt.Println("[MongoDB] Connection closed")
		return Val{Type: 3, Bool: true}, nil
	}
	return Val{}, fmt.Errorf("undefined method: %s on MongoDBConnection", method)
}

func scanSQLRows(rows *sql.Rows) (Val, error) {
	cols, err := rows.Columns()
	if err != nil {
		return Val{}, err
	}
	var results []Val
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}
		if err := rows.Scan(columnPointers...); err != nil {
			return Val{}, err
		}
		
		var rowPairs []Val
		for i, colName := range cols {
			val := columns[i]
			var phxVal Val
			if val == nil {
				phxVal = Val{}
			} else {
				switch v := val.(type) {
				case int64:
					phxVal = NewInt(v)
				case int:
					phxVal = NewInt(int64(v))
				case float64:
					phxVal = NewFloat(v)
				case bool:
					phxVal = NewBool(v)
				case []byte:
					phxVal = NewStr(string(v))
				case string:
					phxVal = NewStr(v)
				default:
					phxVal = NewStr(fmt.Sprintf("%v", v))
				}
			}
			rowPairs = append(rowPairs, NewStr(colName), phxVal)
		}
		results = append(results, NewAssociativeArray(rowPairs...))
	}
	return NewArray(results...), nil
}

func NewObject(className string, args ...Val) Val {
	if className == "HttpServer" || className == "PHX\\HttpServer" || className == "HTTPServer" || className == "PHX\\HTTPServer" {
		return Val{Type: 9, Obj: &HTTPServerObject{}}
	}
	if className == "Router" || className == "PHX\\Router" {
		return Val{Type: 9, Obj: &RouterObject{}}
	}
	if className == "Cache" || className == "PHX\\Cache" {
		return Val{Type: 9, Obj: globalCache}
	}
	if className == "File" || className == "PHX\\File" {
		return Val{Type: 9, Obj: &FileObject{}}
	}
	if className == "MySQL" || className == "PHX\\MySQL" {
		return Val{Type: 9, Obj: &MySQLObject{}}
	}
	if className == "PostgreSQL" || className == "PHX\\PostgreSQL" || className == "Postgres" || className == "PHX\\Postgres" {
		return Val{Type: 9, Obj: &PostgresObject{}}
	}
	if className == "MongoDB" || className == "PHX\\MongoDB" || className == "Mongo" || className == "PHX\\Mongo" {
		return Val{Type: 9, Obj: &MongoObject{collections: make(map[string][]Val)}}
	}
	if className == "Exception" || className == "PHX\\Exception" {
		msg := ""
		if len(args) > 0 {
			msg = ToString(args[0]).Str
		}
		return Val{Type: 9, Obj: &PHXExceptionObject{Message: msg}}
	}
	
	obj := &PHXObject{
		ClassName: className,
		Fields:    make(map[string]Val),
		Methods:   make(map[string]Val),
	}
	if methods, ok := classes[className]; ok {
		for k, v := range methods {
			obj.Methods[k] = v
		}
	}
	
	val := Val{Type: 9, Obj: obj}
	if construct, ok := obj.Methods["__construct"]; ok {
		constructArgs := append([]Val{val}, args...)
		construct.Func(constructArgs...)
	}
	return val
}

func CallMethod(obj Val, method string, args ...Val) Val {
	if obj.Type != 9 || obj.Obj == nil {
		panic(fmt.Sprintf("Call to method %s on non-object value", method))
	}
	res, err := obj.Obj.Call(method, args...)
	if err != nil {
		panic(PHXException{Value: Val{Type: 9, Obj: &PHXExceptionObject{Message: err.Error()}}})
	}
	return res
}

func GetProperty(obj Val, prop string) Val {
	if obj.Type != 9 || obj.Obj == nil {
		return Val{}
	}
	return obj.Obj.Get(prop)
}

func SetProperty(obj Val, prop string, val Val) Val {
	if obj.Type != 9 || obj.Obj == nil {
		panic(fmt.Sprintf("Attempt to set property %s on non-object", prop))
	}
	obj.Obj.Set(prop, val)
	return val
}

func SetPropertyIndex(obj Val, prop string, idx Val, val Val) Val {
	arr := GetProperty(obj, prop)
	SetIndex(&arr, idx, val)
	SetProperty(obj, prop, arr)
	return val
}

func PostIncProperty(obj Val, prop string) Val {
	old := GetProperty(obj, prop)
	val := old
	if val.Type == 1 {
		val.Int++
	} else if val.Type == 2 {
		val.Float++
	} else {
		val.Int++
	}
	SetProperty(obj, prop, val)
	return old
}

func PostDecProperty(obj Val, prop string) Val {
	old := GetProperty(obj, prop)
	val := old
	if val.Type == 1 {
		val.Int--
	} else if val.Type == 2 {
		val.Float--
	} else {
		val.Int--
	}
	SetProperty(obj, prop, val)
	return old
}

func PreIncProperty(obj Val, prop string) Val {
	val := GetProperty(obj, prop)
	if val.Type == 1 {
		val.Int++
	} else if val.Type == 2 {
		val.Float++
	} else {
		val.Int++
	}
	SetProperty(obj, prop, val)
	return val
}

func PreDecProperty(obj Val, prop string) Val {
	val := GetProperty(obj, prop)
	if val.Type == 1 {
		val.Int--
	} else if val.Type == 2 {
		val.Float--
	} else {
		val.Int--
	}
	SetProperty(obj, prop, val)
	return val
}

func AssignPropertyOp(obj Val, prop string, op string, val Val) Val {
	old := GetProperty(obj, prop)
	var res Val
	switch op {
	case "+=": res = Add(old, val)
	case "-=": res = Sub(old, val)
	case "*=": res = Mul(old, val)
	}
	SetProperty(obj, prop, res)
	return res
}

func NewInt(v int64) Val { return Val{Type: 1, Int: v} }
func NewFloat(v float64) Val { return Val{Type: 2, Float: v} }
func NewBool(v bool) Val { return Val{Type: 3, Bool: v} }
func NewStr(v string) Val { return Val{Type: 4, Str: v} }

func NewArray(elems ...Val) Val {
	var arr []Val
	for i, el := range elems {
		arr = append(arr, NewInt(int64(i)), el)
	}
	return Val{Type: 5, Array: arr}
}

func NewAssociativeArray(elems ...Val) Val {
	return Val{Type: 5, Array: elems}
}

func Add(a, b Val) Val {
	if a.Type == 1 && b.Type == 1 {
		return Val{Type: 1, Int: a.Int + b.Int}
	}
	if a.Type == 2 || b.Type == 2 {
		return Val{Type: 2, Float: toF(a) + toF(b)}
	}
	return Val{Type: 1, Int: a.Int + b.Int}
}

func Sub(a, b Val) Val {
	if a.Type == 1 && b.Type == 1 {
		return Val{Type: 1, Int: a.Int - b.Int}
	}
	if a.Type == 2 || b.Type == 2 {
		return Val{Type: 2, Float: toF(a) - toF(b)}
	}
	return Val{Type: 1, Int: a.Int - b.Int}
}

func Mul(a, b Val) Val {
	if a.Type == 1 && b.Type == 1 {
		return Val{Type: 1, Int: a.Int * b.Int}
	}
	if a.Type == 2 || b.Type == 2 {
		return Val{Type: 2, Float: toF(a) * toF(b)}
	}
	return Val{Type: 1, Int: a.Int * b.Int}
}

func Div(a, b Val) Val {
	if a.Type == 1 && b.Type == 1 {
		if b.Int == 0 {
			panic("Division by zero")
		}
		if a.Int % b.Int == 0 {
			return Val{Type: 1, Int: a.Int / b.Int}
		}
		return Val{Type: 2, Float: float64(a.Int) / float64(b.Int)}
	}
	lVal := toF(a)
	rVal := toF(b)
	if rVal == 0 {
		panic("Division by zero")
	}
	return Val{Type: 2, Float: lVal / rVal}
}

func Mod(a, b Val) Val {
	if a.Type == 1 && b.Type == 1 {
		if b.Int == 0 {
			panic("Division by zero")
		}
		return Val{Type: 1, Int: a.Int % b.Int}
	}
	if b.Int == 0 {
		panic("Division by zero")
	}
	return NewInt(a.Int % b.Int)
}

func Lt(a, b Val) Val {
	if a.Type == 1 && b.Type == 1 {
		return Val{Type: 3, Bool: a.Int < b.Int}
	}
	if a.Type == 2 || b.Type == 2 {
		return Val{Type: 3, Bool: toF(a) < toF(b)}
	}
	return Val{Type: 3, Bool: a.Int < b.Int}
}

func Gt(a, b Val) Val {
	if a.Type == 1 && b.Type == 1 {
		return Val{Type: 3, Bool: a.Int > b.Int}
	}
	if a.Type == 2 || b.Type == 2 {
		return Val{Type: 3, Bool: toF(a) > toF(b)}
	}
	return Val{Type: 3, Bool: a.Int > b.Int}
}

func Le(a, b Val) Val {
	if a.Type == 1 && b.Type == 1 {
		return Val{Type: 3, Bool: a.Int <= b.Int}
	}
	if a.Type == 2 || b.Type == 2 {
		return Val{Type: 3, Bool: toF(a) <= toF(b)}
	}
	return Val{Type: 3, Bool: a.Int <= b.Int}
}

func Ge(a, b Val) Val {
	if a.Type == 1 && b.Type == 1 {
		return Val{Type: 3, Bool: a.Int >= b.Int}
	}
	if a.Type == 2 || b.Type == 2 {
		return Val{Type: 3, Bool: toF(a) >= toF(b)}
	}
	return Val{Type: 3, Bool: a.Int >= b.Int}
}

func Eq(a, b Val) Val {
	if a.Type == 0 || b.Type == 0 {
		return Val{Type: 3, Bool: a.Type == 0 && b.Type == 0}
	}
	if a.Type == 3 && b.Type == 3 {
		return Val{Type: 3, Bool: a.Bool == b.Bool}
	}
	if a.Type == 1 && b.Type == 1 {
		return Val{Type: 3, Bool: a.Int == b.Int}
	}
	if a.Type == 4 && b.Type == 4 {
		return Val{Type: 3, Bool: a.Str == b.Str}
	}
	if a.Type == 2 || b.Type == 2 {
		return Val{Type: 3, Bool: toF(a) == toF(b)}
	}
	if a.Type != b.Type {
		return Val{Type: 3, Bool: false}
	}
	return Val{Type: 3, Bool: a.Int == b.Int}
}

func Ne(a, b Val) Val {
	return NewBool(!Eq(a, b).Bool)
}

func Not(a Val) Val {
	return NewBool(!IsTruthy(a))
}

func Neg(a Val) Val {
	if a.Type == 2 {
		return NewFloat(-a.Float)
	}
	return NewInt(-a.Int)
}

func AddAssign(a *Val, b Val) Val {
	*a = Add(*a, b)
	return *a
}

func SubAssign(a *Val, b Val) Val {
	*a = Sub(*a, b)
	return *a
}

func MulAssign(a *Val, b Val) Val {
	*a = Mul(*a, b)
	return *a
}

func PrefixInc(a *Val) Val {
	if a.Type == 1 {
		a.Int++
		return *a
	}
	if a.Type == 2 {
		a.Float++
	} else {
		a.Int++
	}
	return *a
}

func PrefixDec(a *Val) Val {
	if a.Type == 1 {
		a.Int--
		return *a
	}
	if a.Type == 2 {
		a.Float--
	} else {
		a.Int--
	}
	return *a
}

func PostfixInc(a *Val) Val {
	old := *a
	if a.Type == 1 {
		a.Int++
		return old
	}
	if a.Type == 2 {
		a.Float++
	} else {
		a.Int++
	}
	return old
}

func PostfixDec(a *Val) Val {
	old := *a
	if a.Type == 1 {
		a.Int--
		return old
	}
	if a.Type == 2 {
		a.Float--
	} else {
		a.Int--
	}
	return old
}

func IsTruthy(v Val) bool {
	if v.Type == 1 {
		return v.Int != 0
	}
	if v.Type == 3 {
		return v.Bool
	}
	switch v.Type {
	case 0: return false
	case 2: return v.Float != 0
	case 4: return v.Str != "" && v.Str != "0"
	default: return true
	}
}

func GetIndex(arr Val, idx Val) Val {
	if arr.Type == 5 {
		for i := 0; i < len(arr.Array); i += 2 {
			if i+1 < len(arr.Array) && Eq(arr.Array[i], idx).Bool {
				return arr.Array[i+1]
			}
		}
	}
	return Val{}
}

func SetIndex(arr *Val, idx Val, val Val) Val {
	if arr.Type != 5 {
		arr.Type = 5
		arr.Array = nil
	}
	if idx.Type == 0 {
		maxKey := int64(-1)
		for i := 0; i < len(arr.Array); i += 2 {
			if arr.Array[i].Type == 1 && arr.Array[i].Int > maxKey {
				maxKey = arr.Array[i].Int
			}
		}
		arr.Array = append(arr.Array, NewInt(maxKey+1), val)
		return val
	}
	for i := 0; i < len(arr.Array); i += 2 {
		if i+1 < len(arr.Array) && Eq(arr.Array[i], idx).Bool {
			arr.Array[i+1] = val
			return val
		}
	}
	arr.Array = append(arr.Array, idx, val)
	return val
}

func toF(v Val) float64 {
	switch v.Type {
	case 1: return float64(v.Int)
	case 2: return v.Float
	default: return 0
	}
}

func ToString(v Val) Val {
	switch v.Type {
	case 0: return NewStr("null")
	case 1: return NewStr(fmt.Sprintf("%d", v.Int))
	case 2: return NewStr(fmt.Sprintf("%g", v.Float))
	case 3: return NewStr(fmt.Sprintf("%t", v.Bool))
	case 4: return v
	default: return NewStr("<Object>")
	}
}

func Concat(parts ...Val) Val {
	var sb strings.Builder
	for _, p := range parts {
		sb.WriteString(ToString(p).Str)
	}
	return NewStr(sb.String())
}

func echo(args ...Val) {
	for _, arg := range args {
		switch arg.Type {
		case 0:
		case 1: fmt.Print(arg.Int)
		case 2: fmt.Print(arg.Float)
		case 3: fmt.Print(arg.Bool)
		case 4: fmt.Print(arg.Str)
		default: fmt.Print("<Object>")
		}
	}
}

func Call(fn Val, args ...Val) Val {
	if fn.Type != 8 || fn.Func == nil {
		panic("call to non-function value")
	}
	return fn.Func(args...)
}

// Builtins
var count = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) == 0 { return NewInt(0) }
	arr := args[0]
	if arr.Type == 5 {
		return NewInt(int64(len(arr.Array) / 2))
	}
	return NewInt(0)
}}

var microtime = Val{Type: 8, Func: func(args ...Val) Val {
	return NewFloat(float64(time.Now().UnixNano()) / 1e9)
}}

var channel = Val{Type: 8, Func: func(args ...Val) Val {
	cap := int(args[0].Int)
	return Val{Type: 6, Chan: make(chan Val, cap)}
}}

var send = Val{Type: 8, Func: func(args ...Val) Val {
	ch := args[0].Chan
	val := args[1]
	ch <- val
	return Val{}
}}

var receive = Val{Type: 8, Func: func(args ...Val) Val {
	ch := args[0].Chan
	return <-ch
}}

type ThreadObject struct {
	done chan struct{}
	val  Val
}

func (t *ThreadObject) Get(prop string) Val { return Val{} }
func (t *ThreadObject) Set(prop string, val Val) {}
func (t *ThreadObject) Call(method string, args ...Val) (Val, error) {
	if method == "join" {
		<-t.done
		return t.val, nil
	}
	return Val{}, fmt.Errorf("undefined method: %s on Thread", method)
}

var spawn = Val{Type: 8, Func: func(args ...Val) Val {
	fn := args[0]
	t := &ThreadObject{
		done: make(chan struct{}),
	}
	go func() {
		defer close(t.done)
		t.val = fn.Func(args[1:]...)
	}()
	return Val{Type: 9, Obj: t}
}}

var sleep = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) == 0 { return Val{} }
	d := toF(args[0])
	time.Sleep(time.Duration(d * float64(time.Second)))
	return Val{}
}}

func formatPrintR(v Val, indent string) string {
	switch v.Type {
	case 5:
		var sb strings.Builder
		sb.WriteString("Array\n" + indent + "(\n")
		nextIndent := indent + "    "
		for i := 0; i < len(v.Array); i += 2 {
			keyStr := ToString(v.Array[i]).Str
			valStr := formatPrintR(v.Array[i+1], nextIndent)
			sb.WriteString(fmt.Sprintf("%s[%s] => %s\n", nextIndent, keyStr, valStr))
		}
		sb.WriteString(indent + ")")
		return sb.String()
	default:
		return ToString(v).Str
	}
}

var print_r = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) == 0 { return Val{} }
	fmt.Println(formatPrintR(args[0], ""))
	return Val{}
}}

var strtoupper = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) == 0 { return NewStr("") }
	return NewStr(strings.ToUpper(ToString(args[0]).Str))
}}

var strtolower = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) == 0 { return NewStr("") }
	return NewStr(strings.ToLower(ToString(args[0]).Str))
}}

var str_repeat = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) < 2 { return NewStr("") }
	s := ToString(args[0]).Str
	c := int(args[1].Int)
	if c < 0 { c = 0 }
	return NewStr(strings.Repeat(s, c))
}}

var trim = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) == 0 { return NewStr("") }
	return NewStr(strings.TrimSpace(ToString(args[0]).Str))
}}

var str_replace = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) < 3 { return NewStr("") }
	search := ToString(args[0]).Str
	replace := ToString(args[1]).Str
	subject := ToString(args[2]).Str
	return NewStr(strings.ReplaceAll(subject, search, replace))
}}

var number_format = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) == 0 { return NewStr("") }
	decimals := 0
	if len(args) >= 2 {
		decimals = int(args[1].Int)
	}
	val := toF(args[0])
	return NewStr(fmt.Sprintf("%.*f", decimals, val))
}}

var strlen = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) == 0 { return NewInt(0) }
	return NewInt(int64(len(ToString(args[0]).Str)))
}}

var strpos = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) < 2 { return Val{Type: 3, Bool: false} }
	h := ToString(args[0]).Str
	n := ToString(args[1]).Str
	idx := strings.Index(h, n)
	if idx == -1 {
		return Val{Type: 3, Bool: false}
	}
	return NewInt(int64(idx))
}}

var substr = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) < 2 { return NewStr("") }
	s := ToString(args[0]).Str
	start := int(args[1].Int)
	length := len(s)
	if len(args) >= 3 {
		length = int(args[2].Int)
	}
	
	runes := []rune(s)
	n := len(runes)
	
	if start < 0 {
		start = n + start
		if start < 0 { start = 0 }
	}
	if start > n {
		return NewStr("")
	}
	
	end := start + length
	if length < 0 {
		end = n + length
	}
	if end < start {
		end = start
	}
	if end > n {
		end = n
	}
	return NewStr(string(runes[start:end]))
}}

var reader = bufio.NewReader(os.Stdin)

var readline = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) > 0 {
		fmt.Print(ToString(args[0]).Str)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return NewStr("")
	}
	line = strings.TrimRight(line, "\r\n")
	return NewStr(line)
}}

var intval = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) == 0 { return NewInt(0) }
	v := args[0]
	switch v.Type {
	case 1: return v
	case 2: return NewInt(int64(v.Float))
	case 3:
		if v.Bool { return NewInt(1) }
		return NewInt(0)
	case 4:
		var i int64
		fmt.Sscanf(v.Str, "%d", &i)
		return NewInt(i)
	default: return NewInt(0)
	}
}}

var floatval = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) == 0 { return NewFloat(0) }
	v := args[0]
	switch v.Type {
	case 1: return NewFloat(float64(v.Int))
	case 2: return v
	case 3:
		if v.Bool { return NewFloat(1) }
		return NewFloat(0)
	case 4:
		var f float64
		fmt.Sscanf(v.Str, "%f", &f)
		return NewFloat(f)
	default: return NewFloat(0)
	}
}}

var strval = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) == 0 { return NewStr("") }
	return ToString(args[0])
}}

var abs = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) == 0 { return NewInt(0) }
	v := args[0]
	if v.Type == 2 {
		if v.Float < 0 { return NewFloat(-v.Float) }
		return v
	}
	if v.Int < 0 { return NewInt(-v.Int) }
	return v
}}

var min = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) == 0 { return Val{} }
	var list []Val
	if len(args) == 1 && args[0].Type == 5 {
		for i := 1; i < len(args[0].Array); i += 2 {
			list = append(list, args[0].Array[i])
		}
	} else {
		list = args
	}
	if len(list) == 0 { return Val{} }
	res := list[0]
	for i := 1; i < len(list); i++ {
		if IsTruthy(Lt(list[i], res)) {
			res = list[i]
		}
	}
	return res
}}

var max = Val{Type: 8, Func: func(args ...Val) Val {
	if len(args) == 0 { return Val{} }
	var list []Val
	if len(args) == 1 && args[0].Type == 5 {
		for i := 1; i < len(args[0].Array); i += 2 {
			list = append(list, args[0].Array[i])
		}
	} else {
		list = args
	}
	if len(list) == 0 { return Val{} }
	res := list[0]
	for i := 1; i < len(list); i++ {
		if IsTruthy(Gt(list[i], res)) {
			res = list[i]
		}
	}
	return res
}}

var rand = Val{Type: 8, Func: func(args ...Val) Val {
	minVal := int64(0)
	maxVal := int64(2147483647)
	if len(args) >= 2 {
		minVal = args[0].Int
		maxVal = args[1].Int
	}
	if minVal >= maxVal {
		return NewInt(minVal)
	}
	n := time.Now().UnixNano()
	r := minVal + (n % (maxVal - minVal + 1))
	return NewInt(r)
}}

var intdiv = Val{Type: 8, Func: func(args ...Val) Val {
	return NewInt(args[0].Int / args[1].Int)
}}

func valToJSON(v Val) ([]byte, error) {
	switch v.Type {
	case 0:
		return []byte("null"), nil
	case 1:
		return json.Marshal(v.Int)
	case 2:
		return json.Marshal(v.Float)
	case 3:
		return json.Marshal(v.Bool)
	case 4:
		return json.Marshal(v.Str)
	case 5: // Array
		isSequential := true
		for i := 0; i < len(v.Array); i += 2 {
			if v.Array[i].Type != 1 || v.Array[i].Int != int64(i/2) {
				isSequential = false
				break
			}
		}
		if isSequential {
			var list []interface{}
			for i := 1; i < len(v.Array); i += 2 {
				raw, err := valToInterface(v.Array[i])
				if err != nil {
					return nil, err
				}
				list = append(list, raw)
			}
			return json.Marshal(list)
		} else {
			m := make(map[string]interface{})
			for i := 0; i < len(v.Array); i += 2 {
				key := ToString(v.Array[i]).Str
				val, err := valToInterface(v.Array[i+1])
				if err != nil {
					return nil, err
				}
				m[key] = val
			}
			return json.Marshal(m)
		}
	default:
		return []byte("\"<Object>\""), nil
	}
}

func valToInterface(v Val) (interface{}, error) {
	switch v.Type {
	case 0:
		return nil, nil
	case 1:
		return v.Int, nil
	case 2:
		return v.Float, nil
	case 3:
		return v.Bool, nil
	case 4:
		return v.Str, nil
	case 5:
		isSequential := true
		for i := 0; i < len(v.Array); i += 2 {
			if v.Array[i].Type != 1 || v.Array[i].Int != int64(i/2) {
				isSequential = false
				break
			}
		}
		if isSequential {
			var list []interface{}
			for i := 1; i < len(v.Array); i += 2 {
				raw, err := valToInterface(v.Array[i])
				if err != nil {
					return nil, err
				}
				list = append(list, raw)
			}
			return list, nil
		} else {
			m := make(map[string]interface{})
			for i := 0; i < len(v.Array); i += 2 {
				key := ToString(v.Array[i]).Str
				val, err := valToInterface(v.Array[i+1])
				if err != nil {
					return nil, err
				}
				m[key] = val
			}
			return m, nil
		}
	default:
		return "<Object>", nil
	}
}

func getMapVal(arr Val, key string) Val {
	if arr.Type != 5 {
		return Val{}
	}
	for i := 0; i < len(arr.Array); i += 2 {
		if ToString(arr.Array[i]).Str == key {
			return arr.Array[i+1]
		}
	}
	return Val{}
}

type Route struct {
	method     string
	pattern    *regexp.Regexp
	paramNames []string
	handler    Val
}

type RouterObject struct {
	routes        []Route
	currentPrefix string
}

func (r *RouterObject) Get(prop string) Val { return Val{} }
func (r *RouterObject) Set(prop string, val Val) {}
func (r *RouterObject) Call(method string, args ...Val) (Val, error) {
	switch method {
	case "get", "post", "put", "delete":
		if len(args) < 2 {
			return Val{}, fmt.Errorf("Router::%s expects at least 2 arguments: path and handler", method)
		}
		path := ToString(args[0]).Str
		handler := args[1]
		r.addRoute(strings.ToUpper(method), path, handler)
		return Val{Type: 9, Obj: r}, nil

	case "group":
		if len(args) < 2 {
			return Val{}, fmt.Errorf("Router::group expects 2 arguments: prefix and callback")
		}
		prefix := ToString(args[0]).Str
		callback := args[1]
		
		oldPrefix := r.currentPrefix
		r.currentPrefix = oldPrefix + prefix
		
		Call(callback, Val{Type: 9, Obj: r})
		
		r.currentPrefix = oldPrefix
		return Val{Type: 9, Obj: r}, nil
	}
	return Val{}, fmt.Errorf("undefined method: %s on Router", method)
}

func (r *RouterObject) addRoute(method, path string, handler Val) {
	fullPath := r.currentPrefix + path
	rePattern := "^"
	var paramNames []string
	
	parts := strings.Split(fullPath, "/")
	for idx, part := range parts {
		if idx > 0 {
			rePattern += "/"
		}
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			paramName := part[1 : len(part)-1]
			rePattern += fmt.Sprintf("(?P<%s>[^/]+)", paramName)
			paramNames = append(paramNames, paramName)
		} else {
			rePattern += regexp.QuoteMeta(part)
		}
	}
	rePattern += "$"
	
	re := regexp.MustCompile(rePattern)
	r.routes = append(r.routes, Route{
		method:     method,
		pattern:    re,
		paramNames: paramNames,
		handler:    handler,
	})
}

type CacheItem struct {
	value      Val
	expiration time.Time
}

type CacheObject struct {
	mu    sync.RWMutex
	items map[string]CacheItem
}

var globalCache = &CacheObject{items: make(map[string]CacheItem)}

func (c *CacheObject) Get(prop string) Val { return Val{} }
func (c *CacheObject) Set(prop string, val Val) {}
func (c *CacheObject) Call(method string, args ...Val) (Val, error) {
	switch method {
	case "get":
		if len(args) < 1 {
			return Val{}, fmt.Errorf("Cache::get expects 1 argument")
		}
		key := ToString(args[0]).Str
		
		c.mu.RLock()
		item, exists := c.items[key]
		c.mu.RUnlock()
		
		if !exists {
			return Val{}, nil
		}
		if !item.expiration.IsZero() && time.Now().After(item.expiration) {
			c.mu.Lock()
			delete(c.items, key)
			c.mu.Unlock()
			return Val{}, nil
		}
		return item.value, nil

	case "set":
		if len(args) < 2 {
			return Val{}, fmt.Errorf("Cache::set expects at least 2 arguments")
		}
		key := ToString(args[0]).Str
		val := args[1]
		ttl := int64(0)
		if len(args) > 2 {
			ttl = args[2].Int
		}
		
		var exp time.Time
		if ttl > 0 {
			exp = time.Now().Add(time.Duration(ttl) * time.Second)
		}
		
		c.mu.Lock()
		c.items[key] = CacheItem{
			value:      val,
			expiration: exp,
		}
		c.mu.Unlock()
		return Val{Type: 3, Bool: true}, nil

	case "delete":
		if len(args) < 1 {
			return Val{}, fmt.Errorf("Cache::delete expects 1 argument")
		}
		key := ToString(args[0]).Str
		
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return Val{Type: 3, Bool: true}, nil
	}
	return Val{}, fmt.Errorf("undefined method: %s on Cache", method)
}

type HTTPRequestObject struct {
	req        *http.Request
	body       string
	pathParams map[string]string
}

func (r *HTTPRequestObject) Get(prop string) Val { return Val{} }
func (r *HTTPRequestObject) Set(prop string, val Val) {}
func (r *HTTPRequestObject) Call(method string, args ...Val) (Val, error) {
	switch method {
	case "getMethod":
		return NewStr(r.req.Method), nil
	case "getUri":
		return NewStr(r.req.RequestURI), nil
	case "getPath":
		return NewStr(r.req.URL.Path), nil
	case "getBody":
		return NewStr(r.body), nil
	case "getHeaders":
		var pairs []Val
		for name, values := range r.req.Header {
			if len(values) > 0 {
				pairs = append(pairs, NewStr(name), NewStr(values[0]))
			}
		}
		return NewAssociativeArray(pairs...), nil
	case "getQuery":
		var pairs []Val
		for name, values := range r.req.URL.Query() {
			if len(values) > 0 {
				pairs = append(pairs, NewStr(name), NewStr(values[0]))
			}
		}
		return NewAssociativeArray(pairs...), nil
	case "getPost":
		_ = r.req.ParseForm()
		var pairs []Val
		for name, values := range r.req.PostForm {
			if len(values) > 0 {
				pairs = append(pairs, NewStr(name), NewStr(values[0]))
			}
		}
		return NewAssociativeArray(pairs...), nil
	case "getParam":
		if len(args) < 1 {
			return Val{}, fmt.Errorf("HTTPRequest::getParam expects 1 argument")
		}
		paramName := ToString(args[0]).Str
		val, exists := r.pathParams[paramName]
		if !exists {
			return Val{}, nil
		}
		return NewStr(val), nil
	}
	return Val{}, fmt.Errorf("undefined method: %s on HTTPRequest", method)
}

type HTTPResponseObject struct {
	w             http.ResponseWriter
	status        int
	ended         bool
	headerWritten bool
}

func (r *HTTPResponseObject) writeHeader() {
	if !r.headerWritten {
		r.w.WriteHeader(r.status)
		r.headerWritten = true
	}
}

func (r *HTTPResponseObject) Get(prop string) Val { return Val{} }
func (r *HTTPResponseObject) Set(prop string, val Val) {}
func (r *HTTPResponseObject) Call(method string, args ...Val) (Val, error) {
	switch method {
	case "status":
		if len(args) < 1 {
			return Val{}, fmt.Errorf("HTTPResponse::status expects 1 argument")
		}
		r.status = int(args[0].Int)
		return Val{Type: 9, Obj: r}, nil

	case "header":
		if len(args) < 2 {
			return Val{}, fmt.Errorf("HTTPResponse::header expects 2 arguments")
		}
		name := ToString(args[0]).Str
		value := ToString(args[1]).Str
		r.w.Header().Set(name, value)
		return Val{Type: 9, Obj: r}, nil

	case "write":
		if len(args) < 1 {
			return Val{}, fmt.Errorf("HTTPResponse::write expects 1 argument")
		}
		if r.ended {
			return Val{}, fmt.Errorf("HTTPResponse already ended")
		}
		r.writeHeader()
		content := ToString(args[0]).Str
		r.w.Write([]byte(content))
		return Val{Type: 9, Obj: r}, nil

	case "end":
		if r.ended {
			return Val{}, nil
		}
		r.writeHeader()
		if len(args) > 0 {
			content := ToString(args[0]).Str
			r.w.Write([]byte(content))
		}
		r.ended = true
		return Val{}, nil

	case "json":
		if len(args) < 1 {
			return Val{}, fmt.Errorf("HTTPResponse::json expects 1 argument")
		}
		if r.ended {
			return Val{}, fmt.Errorf("HTTPResponse already ended")
		}
		jsonVal := args[0]
		jsonData, err := valToJSON(jsonVal)
		if err != nil {
			return Val{}, err
		}
		r.w.Header().Set("Content-Type", "application/json")
		r.writeHeader()
		r.w.Write(jsonData)
		r.ended = true
		return Val{}, nil
	}
	return Val{}, fmt.Errorf("undefined method: %s on HTTPResponse", method)
}

type HTTPServerObject struct {
	server        *http.Server
	config        Val
	middlewares   []Val
	router        *RouterObject
	dbPools       map[string]*sql.DB
	dbPoolsDriver map[string]string
	cacheClient   *CacheObject
}

func (s *HTTPServerObject) Get(prop string) Val { return Val{} }
func (s *HTTPServerObject) Set(prop string, val Val) {}
func (s *HTTPServerObject) Call(method string, args ...Val) (Val, error) {
	if s.dbPools == nil {
		s.dbPools = make(map[string]*sql.DB)
		s.dbPoolsDriver = make(map[string]string)
	}
	switch method {
	case "configure":
		if len(args) < 1 {
			return Val{}, fmt.Errorf("HTTPServer::configure expects 1 argument")
		}
		s.config = args[0]
		return Val{Type: 9, Obj: s}, nil

	case "use":
		if len(args) < 1 {
			return Val{}, fmt.Errorf("HTTPServer::use expects 1 argument")
		}
		s.middlewares = append(s.middlewares, args[0])
		return Val{Type: 9, Obj: s}, nil

	case "router":
		if s.router == nil {
			s.router = &RouterObject{}
		}
		return Val{Type: 9, Obj: s.router}, nil

	case "database":
		if len(args) < 1 {
			return Val{}, fmt.Errorf("HTTPServer::database expects 1 argument")
		}
		config := args[0]
		if config.Type != 5 {
			return Val{}, fmt.Errorf("HTTPServer::database config must be an array")
		}
		for i := 0; i < len(config.Array); i += 2 {
			poolName := ToString(config.Array[i]).Str
			dbConfig := config.Array[i+1]
			if dbConfig.Type != 5 {
				continue
			}
			driver := getMapVal(dbConfig, "driver").Str
			host := getMapVal(dbConfig, "host").Str
			user := getMapVal(dbConfig, "user").Str
			password := getMapVal(dbConfig, "password").Str
			dbName := getMapVal(dbConfig, "database").Str
			if dbName == "" {
				dbName = getMapVal(dbConfig, "dbname").Str
			}
			if user == "" {
				user = "root"
			}
			
			var dsn string
			var dbConn *sql.DB
			var err error
			
			if driver == "mysql" {
				dsn = fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", user, password, host, dbName)
				dbConn, err = sql.Open("mysql", dsn)
			} else if driver == "postgres" || driver == "postgresql" {
				dsn = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, password, host, dbName)
				dbConn, err = sql.Open("postgres", dsn)
			} else {
				return Val{}, fmt.Errorf("unsupported database driver: %s", driver)
			}
			
			if err != nil {
				return Val{}, err
			}
			
			poolSize := int64(10)
			poolSizeVal := getMapVal(dbConfig, "pool_size")
			if poolSizeVal.Type == 1 {
				poolSize = poolSizeVal.Int
			}
			dbConn.SetMaxOpenConns(int(poolSize))
			dbConn.SetMaxIdleConns(int(poolSize))
			
			s.dbPools[poolName] = dbConn
			s.dbPoolsDriver[poolName] = driver
		}
		return Val{Type: 9, Obj: s}, nil

	case "db":
		name := "default"
		if len(args) > 0 {
			name = ToString(args[0]).Str
		}
		conn, exists := s.dbPools[name]
		if !exists {
			return Val{}, fmt.Errorf("database pool %s not configured", name)
		}
		driver := s.dbPoolsDriver[name]
		if driver == "mysql" {
			return Val{Type: 9, Obj: &MySQLObject{dbConn: conn, connected: true}}, nil
		} else {
			return Val{Type: 9, Obj: &PostgresObject{dbConn: conn, connected: true}}, nil
		}

	case "cache":
		if s.cacheClient == nil {
			s.cacheClient = globalCache
		}
		if len(args) > 0 {
			return Val{Type: 9, Obj: s}, nil
		}
		return Val{Type: 9, Obj: s.cacheClient}, nil

	case "listen":
		if len(args) < 1 {
			return Val{}, fmt.Errorf("HTTPServer::listen expects at least 1 argument: port")
		}
		port := int(args[0].Int)
		
		mux := http.NewServeMux()
		
		if len(args) >= 2 && (args[1].Type == 8 && args[1].Func != nil) {
			callback := args[1]
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				var bodyBytes []byte
				if r.Body != nil {
					bodyBytes, _ = io.ReadAll(r.Body)
					r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				}
				reqObj := &HTTPRequestObject{
					req:        r,
					body:       string(bodyBytes),
					pathParams: make(map[string]string),
				}
				resObj := &HTTPResponseObject{
					w:      w,
					status: http.StatusOK,
				}
				callback.Func(Val{Type: 9, Obj: reqObj}, Val{Type: 9, Obj: resObj})
				if !resObj.ended {
					resObj.writeHeader()
					resObj.ended = true
				}
			})
		} else {
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				var bodyBytes []byte
				if r.Body != nil {
					bodyBytes, _ = io.ReadAll(r.Body)
					r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				}
				reqObj := &HTTPRequestObject{
					req:        r,
					body:       string(bodyBytes),
					pathParams: make(map[string]string),
				}
				resObj := &HTTPResponseObject{
					w:      w,
					status: http.StatusOK,
				}

				var runNext func(idx int, currentReq Val) Val
				runNext = func(idx int, currentReq Val) Val {
					if idx >= len(s.middlewares) {
						return s.dispatchRoute(currentReq, Val{Type: 9, Obj: resObj})
					}

					middleware := s.middlewares[idx]
					nextFunc := Val{Type: 8, Func: func(args ...Val) Val {
						nextReq := currentReq
						if len(args) > 0 {
							nextReq = args[0]
						}
						return runNext(idx+1, nextReq)
					}}

					resVal := Val{Type: 9, Obj: resObj}
					if middleware.Type == 8 {
						return Call(middleware, currentReq, resVal, nextFunc)
					} else if middleware.Type == 4 {
						obj := NewObject(middleware.Str)
						if obj.Type == 9 && obj.Obj != nil {
							if pObj, ok := obj.Obj.(*PHXObject); ok {
								if _, hasHandle := pObj.Methods["handle"]; hasHandle {
									return CallMethod(obj, "handle", currentReq, resVal, nextFunc)
								}
								if _, hasInvoke := pObj.Methods["__invoke"]; hasInvoke {
									return CallMethod(obj, "__invoke", currentReq, resVal, nextFunc)
								}
							}
						}
					}
					return runNext(idx+1, currentReq)
				}

				_ = runNext(0, Val{Type: 9, Obj: reqObj})

				if !resObj.ended {
					resObj.writeHeader()
					resObj.ended = true
				}
			})
		}

		addr := fmt.Sprintf(":%d", port)
		s.server = &http.Server{
			Addr:    addr,
			Handler: mux,
		}
		
		fmt.Printf("PHX Server listening on http://localhost:%d\n", port)
		err := s.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			return Val{}, err
		}
		return Val{Type: 3, Bool: true}, nil
	}
	return Val{}, fmt.Errorf("undefined method: %s on HTTPServer", method)
}

func (s *HTTPServerObject) dispatchRoute(reqVal, resVal Val) Val {
	reqObj := reqVal.Obj.(*HTTPRequestObject)
	resObj := resVal.Obj.(*HTTPResponseObject)
	
	if s.router == nil {
		resObj.status = http.StatusNotFound
		resObj.ended = true
		resObj.w.WriteHeader(http.StatusNotFound)
		resObj.w.Write([]byte("Not Found - No Router"))
		return Val{}
	}
	
	method := reqObj.req.Method
	path := reqObj.req.URL.Path
	
	for _, route := range s.router.routes {
		if route.method == method && route.pattern.MatchString(path) {
			matches := route.pattern.FindStringSubmatch(path)
			for i, name := range route.pattern.SubexpNames() {
				if i > 0 && name != "" {
					reqObj.pathParams[name] = matches[i]
				}
			}
			
			handler := route.handler
			if handler.Type == 8 {
				var args []Val
				args = append(args, reqVal, resVal)
				for _, pName := range route.paramNames {
					args = append(args, NewStr(reqObj.pathParams[pName]))
				}
				return Call(handler, args...)
			} else if handler.Type == 4 {
				parts := strings.Split(handler.Str, "@")
				if len(parts) == 2 {
					className := parts[0]
					methodName := parts[1]
					obj := NewObject(className)
					var args []Val
					args = append(args, reqVal, resVal)
					for _, pName := range route.paramNames {
						args = append(args, NewStr(reqObj.pathParams[pName]))
					}
					return CallMethod(obj, methodName, args...)
				}
			}
		}
	}
	
	resObj.status = http.StatusNotFound
	resObj.ended = true
	resObj.w.WriteHeader(http.StatusNotFound)
	resObj.w.Write([]byte("Not Found"))
	return Val{}
}

var classes = make(map[string]map[string]Val)
`
