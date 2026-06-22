package vm

import (
	"fmt"
	"io"
	"os"
	"phx/internal/ast"
	"phx/internal/token"
	"phx/internal/lexer"
	"phx/internal/parser"
)

var (
	TRUE  = &Boolean{Value: true}
	FALSE = &Boolean{Value: false}
	NULL  = &Null{}
	BREAK = &Break{}
	CONTINUE = &Continue{}
	VirtualFilesystem = make(map[string]string)
)

func Evaluate(node ast.Node, env *Environment, out io.Writer) Object {
	switch n := node.(type) {

	// Statements
	case *ast.Program:
		return evalProgram(n.Statements, env, out)

	case *ast.BlockStatement:
		return evalBlockStatement(n.Statements, env, out)

	case *ast.ExpressionStatement:
		return Evaluate(n.Expression, env, out)

	case *ast.InlineHTMLStatement:
		_, err := io.WriteString(out, n.Content)
		if err != nil {
			return newError("failed to write HTML output: %s", err.Error())
		}
		return NULL

	case *ast.EchoStatement:
		for _, expr := range n.Expressions {
			val := Evaluate(expr, env, out)
			if isError(val) {
				return val
			}
			_, err := io.WriteString(out, toString(val))
			if err != nil {
				return newError("failed to write echo output: %s", err.Error())
			}
		}
		return NULL

	case *ast.IfStatement:
		cond := Evaluate(n.Condition, env, out)
		if isError(cond) {
			return cond
		}
		if isTruthy(cond) {
			return Evaluate(n.Consequence, env, out)
		} else if n.Alternative != nil {
			return Evaluate(n.Alternative, env, out)
		}
		return NULL

	case *ast.IncludeStatement:
		val := Evaluate(n.Expression, env, out)
		if isError(val) {
			return val
		}
		pathStr, ok := val.(*String)
		if !ok {
			return newError("include/require filename must be a string")
		}

		var contentBytes []byte
		var err error

		if content, found := VirtualFilesystem[pathStr.Value]; found {
			contentBytes = []byte(content)
		} else {
			contentBytes, err = os.ReadFile(pathStr.Value)
			if err != nil {
				return newError("failed to read include file: %s", err.Error())
			}
		}

		l := lexer.New(string(contentBytes))
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			return newError("parser errors in include file %s: %v", pathStr.Value, p.Errors())
		}

		res := Evaluate(program, env, out)
		if isError(res) {
			return res
		}
		return res

	case *ast.NamespaceStatement:
		env.currentNamespace = n.Name.Literal
		return NULL

	case *ast.UseStatement:
		env.aliases[n.Alias.Literal] = n.Name.Literal
		return NULL

	case *ast.WhileStatement:
		for {
			condition := Evaluate(n.Condition, env, out)
			if isError(condition) {
				return condition
			}
			if !isTruthy(condition) {
				break
			}
			result := Evaluate(n.Body, env, out)
			if result != nil {
				if result.Type() == ERROR_OBJ || result.Type() == RETURN_VALUE_OBJ {
					return result
				}
				if result.Type() == BREAK_OBJ {
					break
				}
				if result.Type() == CONTINUE_OBJ {
					continue
				}
			}
		}
		return NULL

	case *ast.DoWhileStatement:
		for {
			result := Evaluate(n.Body, env, out)
			if result != nil {
				if result.Type() == ERROR_OBJ || result.Type() == RETURN_VALUE_OBJ {
					return result
				}
				if result.Type() == BREAK_OBJ {
					break
				}
				if result.Type() == CONTINUE_OBJ {
					// continue checks condition next
				}
			}
			condition := Evaluate(n.Condition, env, out)
			if isError(condition) {
				return condition
			}
			if !isTruthy(condition) {
				break
			}
		}
		return NULL

	case *ast.ForStatement:
		if n.Init != nil {
			initVal := Evaluate(n.Init, env, out)
			if isError(initVal) {
				return initVal
			}
		}
		for {
			if n.Condition != nil {
				condition := Evaluate(n.Condition, env, out)
				if isError(condition) {
					return condition
				}
				if !isTruthy(condition) {
					break
				}
			}
			result := Evaluate(n.Body, env, out)
			if result != nil {
				if result.Type() == ERROR_OBJ || result.Type() == RETURN_VALUE_OBJ {
					return result
				}
				if result.Type() == BREAK_OBJ {
					break
				}
				if result.Type() == CONTINUE_OBJ {
					// continue executes post-expression next
				}
			}
			if n.Post != nil {
				postVal := Evaluate(n.Post, env, out)
				if isError(postVal) {
					return postVal
				}
			}
		}
		return NULL

	case *ast.BreakStatement:
		return BREAK

	case *ast.ContinueStatement:
		return CONTINUE

	// Expressions
	case *ast.Variable:
		val, ok := env.Get(n.Value)
		if !ok {
			// In PHP, undefined variables return NULL and issue a notice (we just return NULL for now)
			return NULL
		}
		return val

	case *ast.Identifier:
		resolvedName := env.ResolveName(n.Value)
		val, ok := env.Get(resolvedName)
		if !ok {
			return newError("undefined identifier: %s", n.Value)
		}
		return val

	case *ast.AssignExpression:
		val := Evaluate(n.Value, env, out)
		if isError(val) {
			return val
		}

		if n.Token.Type == token.ADD_ASSIGN || n.Token.Type == token.SUB_ASSIGN {
			currVal := getTargetValue(n.Left, env, out)
			if isError(currVal) {
				return currVal
			}
			op := "+"
			if n.Token.Type == token.SUB_ASSIGN {
				op = "-"
			}
			val = evalInfixExpression(op, currVal, val)
			if isError(val) {
				return val
			}
		}

		return setTargetValue(n.Left, val, env, out)

	case *ast.PropertyExpression:
		objVal := Evaluate(n.Object, env, out)
		if isError(objVal) {
			return objVal
		}
		instance, ok := objVal.(*ObjectInstance)
		if !ok {
			return newError("cannot read property of non-object: %s", n.Object.String())
		}
		val, ok := instance.Fields[n.Property.Value]
		if !ok {
			return NULL
		}
		return val

	case *ast.PrefixExpression:
		if n.Operator == "++" || n.Operator == "--" {
			currVal := getTargetValue(n.Right, env, out)
			if isError(currVal) {
				return currVal
			}
			op := "+"
			if n.Operator == "--" {
				op = "-"
			}
			newVal := evalInfixExpression(op, currVal, NewInteger(1))
			if isError(newVal) {
				return newVal
			}
			return setTargetValue(n.Right, newVal, env, out)
		}
		right := Evaluate(n.Right, env, out)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(n.Operator, right)

	case *ast.PostExpression:
		currVal := getTargetValue(n.Left, env, out)
		if isError(currVal) {
			return currVal
		}
		op := "+"
		if n.Operator == "--" {
			op = "-"
		}
		newVal := evalInfixExpression(op, currVal, NewInteger(1))
		if isError(newVal) {
			return newVal
		}
		res := setTargetValue(n.Left, newVal, env, out)
		if isError(res) {
			return res
		}
		return currVal

	case *ast.InfixExpression:
		left := Evaluate(n.Left, env, out)
		if isError(left) {
			return left
		}
		right := Evaluate(n.Right, env, out)
		if isError(right) {
			return right
		}
		return evalInfixExpression(n.Operator, left, right)

	// Literals
	case *ast.IntegerLiteral:
		return NewInteger(n.Value)

	case *ast.FloatLiteral:
		return &Float{Value: n.Value}

	case *ast.StringLiteral:
		if n.Token.Type == token.T_DOUBLE_QUOTED_STRING {
			return interpolateString(n.Value, env, out)
		}
		return &String{Value: unescapeString(n.Value)}

	case *ast.BooleanLiteral:
		return nativeBoolToBooleanObject(n.Value)

	case *ast.NullLiteral:
		return NULL

	case *ast.ReturnStatement:
		val := Evaluate(n.ReturnValue, env, out)
		if isError(val) {
			return val
		}
		return &ReturnValue{Value: val}

	case *ast.FunctionStatement:
		fn := &Function{Parameters: n.Parameters, Body: n.Body, Env: env}
		qualifiedName := env.QualifyName(n.Name.Value)
		env.Set(qualifiedName, fn)
		return NULL

	case *ast.FunctionExpression:
		return &Function{Parameters: n.Parameters, Body: n.Body, Env: env}

	case *ast.ArrayLiteral:
		pairs := []*ArrayPair{}
		var nextIdx int64 = 0
		for _, el := range n.Elements {
			valObj := Evaluate(el.Value, env, out)
			if isError(valObj) {
				return valObj
			}

			var keyObj Object
			if el.Key != nil {
				keyObj = Evaluate(el.Key, env, out)
				if isError(keyObj) {
					return keyObj
				}
				if intKey, ok := keyObj.(*Integer); ok {
					if intKey.Value >= nextIdx {
						nextIdx = intKey.Value + 1
					}
				}
			} else {
				keyObj = NewInteger(nextIdx)
				nextIdx++
			}
			pairs = append(pairs, &ArrayPair{Key: keyObj, Value: valObj})
		}
		return &Array{Pairs: pairs}

	case *ast.IndexExpression:
		leftVal := Evaluate(n.Left, env, out)
		if isError(leftVal) {
			return leftVal
		}
		arr, ok := leftVal.(*Array)
		if !ok {
			if strObj, isStr := leftVal.(*String); isStr {
				if n.Index == nil {
					return newError("cannot use [] for reading from string")
				}
				idxVal := Evaluate(n.Index, env, out)
				if isError(idxVal) {
					return idxVal
				}
				intIdx, ok := idxVal.(*Integer)
				if !ok {
					return newError("string index must be integer")
				}
				if intIdx.Value < 0 || intIdx.Value >= int64(len(strObj.Value)) {
					return NULL
				}
				return &String{Value: string(strObj.Value[intIdx.Value])}
			}
			return newError("cannot read index of non-array: %s", leftVal.Type())
		}
		if n.Index == nil {
			return newError("cannot use [] for reading")
		}
		idxVal := Evaluate(n.Index, env, out)
		if isError(idxVal) {
			return idxVal
		}
		for _, pair := range arr.Pairs {
			if keysEqual(pair.Key, idxVal) {
				return pair.Value
			}
		}
		return NULL

	case *ast.ClassStatement:
		methods := make(map[string]*Function)
		for _, m := range n.Methods {
			methods[m.Name.Value] = &Function{Parameters: m.Parameters, Body: m.Body, Env: env}
		}
		qualifiedName := env.QualifyName(n.Name.Value)
		classObj := &Class{Name: qualifiedName, Methods: methods}
		env.Set(qualifiedName, classObj)
		return NULL

	case *ast.NewExpression:
		resolvedClass := env.ResolveName(n.Class.Value)
		if resolvedClass == "Channel" {
			capacity := 0
			if len(n.Arguments) > 0 {
				argVal := Evaluate(n.Arguments[0], env, out)
				if intVal, ok := argVal.(*Integer); ok {
					capacity = int(intVal.Value)
				}
			}
			return &Channel{Ch: make(chan Object, capacity)}
		}
		classObj, ok := env.Get(resolvedClass)
		if !ok {
			return newError("class not found: %s", n.Class.Value)
		}
		class, ok := classObj.(*Class)
		if !ok {
			return newError("identifier is not a class: %s", n.Class.Value)
		}
		instance := NewObjectInstance(class)
		
		if constructor, hasConstructor := class.Methods["__construct"]; hasConstructor {
			args := []Object{}
			for _, arg := range n.Arguments {
				val := Evaluate(arg, env, out)
				if isError(val) {
					return val
				}
				args = append(args, val)
			}
			
			extendedEnv := NewEnclosedEnvironment(constructor.Env)
			extendedEnv.Set("$this", instance)
			for i, param := range constructor.Parameters {
				if i < len(args) {
					extendedEnv.Set(param.Var.Value, args[i])
				} else if param.DefaultValue != nil {
					val := Evaluate(param.DefaultValue, constructor.Env, out)
					if isError(val) {
						return val
					}
					extendedEnv.Set(param.Var.Value, val)
				} else {
					extendedEnv.Set(param.Var.Value, NULL)
				}
			}
			evaluated := Evaluate(constructor.Body, extendedEnv, out)
			if isError(evaluated) {
				return evaluated
			}
		}
		return instance

	case *ast.CallExpression:
		fnObj := Evaluate(n.Function, env, out)
		if isError(fnObj) {
			return fnObj
		}
		args := []Object{}
		for _, arg := range n.Arguments {
			val := Evaluate(arg, env, out)
			if isError(val) {
				return val
			}
			args = append(args, val)
		}
		if builtin, ok := fnObj.(*Builtin); ok {
			return builtin.Fn(args, env, out)
		}
		fn, ok := fnObj.(*Function)
		if !ok {
			return newError("not a function: %s", n.Function.String())
		}
		return applyFunction(fn, args, out)

	case *ast.MethodCallExpression:
		objVal := Evaluate(n.Object, env, out)
		if isError(objVal) {
			return objVal
		}

		if chanObj, ok := objVal.(*Channel); ok {
			methodName := n.Method.Value
			switch methodName {
			case "send":
				if len(n.Arguments) == 0 {
					return newError("Channel::send expects 1 argument")
				}
				argVal := Evaluate(n.Arguments[0], env, out)
				if isError(argVal) {
					return argVal
				}
				chanObj.Ch <- argVal
				return NULL
			case "recv":
				val, ok := <-chanObj.Ch
				if !ok {
					return NULL
				}
				return val
			case "close":
				close(chanObj.Ch)
				return NULL
			default:
				return newError("undefined method: %s on Channel", methodName)
			}
		}

		if threadObj, ok := objVal.(*Thread); ok {
			methodName := n.Method.Value
			switch methodName {
			case "join":
				<-threadObj.Done
				if threadObj.Err != nil {
					return threadObj.Err
				}
				return threadObj.Val
			default:
				return newError("undefined method: %s on Thread", methodName)
			}
		}

		instance, ok := objVal.(*ObjectInstance)
		if !ok {
			return newError("cannot call method on non-object: %s", n.Object.String())
		}
		method, ok := instance.Class.Methods[n.Method.Value]
		if !ok {
			return newError("undefined method: %s on class %s", n.Method.Value, instance.Class.Name)
		}
		args := []Object{}
		for _, arg := range n.Arguments {
			val := Evaluate(arg, env, out)
			if isError(val) {
				return val
			}
			args = append(args, val)
		}
		
		extendedEnv := NewEnclosedEnvironment(method.Env)
		extendedEnv.Set("$this", instance)
		for i, param := range method.Parameters {
			if i < len(args) {
				extendedEnv.Set(param.Var.Value, args[i])
			} else if param.DefaultValue != nil {
				val := Evaluate(param.DefaultValue, method.Env, out)
				if isError(val) {
					return val
				}
				extendedEnv.Set(param.Var.Value, val)
			} else {
				extendedEnv.Set(param.Var.Value, NULL)
			}
		}
		evaluated := Evaluate(method.Body, extendedEnv, out)
		return unwrapReturnValue(evaluated)

	case *ast.TernaryExpression:
		cond := Evaluate(n.Condition, env, out)
		if isError(cond) {
			return cond
		}
		if isTruthy(cond) {
			return Evaluate(n.Consequence, env, out)
		}
		return Evaluate(n.Alternative, env, out)

	case *ast.SwitchStatement:
		condVal := Evaluate(n.Expr, env, out)
		if isError(condVal) {
			return condVal
		}
		matched := false
		for _, c := range n.Cases {
			caseVal := Evaluate(c.Value, env, out)
			if isError(caseVal) {
				return caseVal
			}
			
			if isEqualLoose(condVal, caseVal) {
				matched = true
				result := Evaluate(c.Body, env, out)
				if result != nil && (result.Type() == RETURN_VALUE_OBJ || result.Type() == ERROR_OBJ) {
					return result
				}
				break
			}
		}
		if !matched && n.Default != nil {
			result := Evaluate(n.Default, env, out)
			if result != nil && (result.Type() == RETURN_VALUE_OBJ || result.Type() == ERROR_OBJ) {
				return result
			}
		}
		return NULL
	}

	return nil
}

func evalProgram(statements []ast.Statement, env *Environment, out io.Writer) Object {
	var result Object
	for _, statement := range statements {
		result = Evaluate(statement, env, out)
		if result != nil {
			if result.Type() == RETURN_VALUE_OBJ {
				return result.(*ReturnValue).Value
			}
			if result.Type() == ERROR_OBJ {
				return result
			}
			if result.Type() == BREAK_OBJ || result.Type() == CONTINUE_OBJ {
				return result
			}
		}
	}
	return result
}

func evalBlockStatement(statements []ast.Statement, env *Environment, out io.Writer) Object {
	var result Object
	for _, statement := range statements {
		result = Evaluate(statement, env, out)
		if result != nil {
			if result.Type() == RETURN_VALUE_OBJ || result.Type() == ERROR_OBJ {
				return result
			}
			if result.Type() == BREAK_OBJ || result.Type() == CONTINUE_OBJ {
				return result
			}
		}
	}
	return result
}

func evalPrefixExpression(operator string, right Object) Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

func evalBangOperatorExpression(right Object) Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NULL:
		return TRUE
	default:
		if isTruthy(right) {
			return FALSE
		}
		return TRUE
	}
}

func evalMinusPrefixOperatorExpression(right Object) Object {
	switch right.Type() {
	case INTEGER_OBJ:
		value := right.(*Integer).Value
		return NewInteger(-value)
	case FLOAT_OBJ:
		value := right.(*Float).Value
		return &Float{Value: -value}
	default:
		return newError("unknown operator: -%s", right.Type())
	}
}

func evalInfixExpression(operator string, left, right Object) Object {
	switch operator {
	case ".":
		return &String{Value: toString(left) + toString(right)}
	case "===":
		return nativeBoolToBooleanObject(isEqualIdentical(left, right))
	case "!==":
		return nativeBoolToBooleanObject(!isEqualIdentical(left, right))
	case "==":
		return nativeBoolToBooleanObject(isEqualLoose(left, right))
	case "!=":
		return nativeBoolToBooleanObject(!isEqualLoose(left, right))
	}

	// Numerical expressions
	if isNumeric(left) && isNumeric(right) {
		return evalNumericInfixExpression(operator, left, right)
	}

	// Relational string comparisons
	if left.Type() == STRING_OBJ && right.Type() == STRING_OBJ {
		lVal := left.(*String).Value
		rVal := right.(*String).Value
		switch operator {
		case "<":
			return nativeBoolToBooleanObject(lVal < rVal)
		case ">":
			return nativeBoolToBooleanObject(lVal > rVal)
		case "<=":
			return nativeBoolToBooleanObject(lVal <= rVal)
		case ">=":
			return nativeBoolToBooleanObject(lVal >= rVal)
		}
	}

	return newError("unknown operator or type mismatch: %s %s %s", left.Type(), operator, right.Type())
}

func evalNumericInfixExpression(operator string, left, right Object) Object {
	isFloatOp := (left.Type() == FLOAT_OBJ || right.Type() == FLOAT_OBJ)

	if isFloatOp {
		lVal := toFloat(left)
		rVal := toFloat(right)
		switch operator {
		case "+":
			return &Float{Value: lVal + rVal}
		case "-":
			return &Float{Value: lVal - rVal}
		case "*":
			return &Float{Value: lVal * rVal}
		case "/":
			if rVal == 0 {
				return newError("Division by zero")
			}
			return &Float{Value: lVal / rVal}
		case "<":
			return nativeBoolToBooleanObject(lVal < rVal)
		case ">":
			return nativeBoolToBooleanObject(lVal > rVal)
		case "<=":
			return nativeBoolToBooleanObject(lVal <= rVal)
		case ">=":
			return nativeBoolToBooleanObject(lVal >= rVal)
		}
	} else {
		lVal := left.(*Integer).Value
		rVal := right.(*Integer).Value
		switch operator {
		case "+":
			return NewInteger(lVal + rVal)
		case "-":
			return NewInteger(lVal - rVal)
		case "*":
			return NewInteger(lVal * rVal)
		case "/":
			if rVal == 0 {
				return newError("Division by zero")
			}
			if lVal%rVal == 0 {
				return NewInteger(lVal / rVal)
			}
			return &Float{Value: float64(lVal) / float64(rVal)}
		case "%":
			if rVal == 0 {
				return newError("Division by zero")
			}
			return NewInteger(lVal % rVal)
		case "<":
			return nativeBoolToBooleanObject(lVal < rVal)
		case ">":
			return nativeBoolToBooleanObject(lVal > rVal)
		case "<=":
			return nativeBoolToBooleanObject(lVal <= rVal)
		case ">=":
			return nativeBoolToBooleanObject(lVal >= rVal)
		}
	}

	return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
}

func nativeBoolToBooleanObject(input bool) *Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func isTruthy(obj Object) bool {
	switch o := obj.(type) {
	case *Null:
		return false
	case *Boolean:
		return o.Value
	case *Integer:
		return o.Value != 0
	case *Float:
		return o.Value != 0.0
	case *String:
		return o.Value != "" && o.Value != "0"
	default:
		return true
	}
}

func isNumeric(obj Object) bool {
	return obj.Type() == INTEGER_OBJ || obj.Type() == FLOAT_OBJ
}

func toFloat(obj Object) float64 {
	switch o := obj.(type) {
	case *Integer:
		return float64(o.Value)
	case *Float:
		return o.Value
	default:
		return 0.0
	}
}

func toString(obj Object) string {
	switch o := obj.(type) {
	case *String:
		return o.Value
	case *Integer:
		return fmt.Sprintf("%d", o.Value)
	case *Float:
		return fmt.Sprintf("%g", o.Value)
	case *Boolean:
		if o.Value {
			return "1"
		}
		return ""
	case *Null:
		return ""
	default:
		return ""
	}
}

func isEqualIdentical(left, right Object) bool {
	if left.Type() != right.Type() {
		return false
	}
	switch l := left.(type) {
	case *Integer:
		r := right.(*Integer)
		return l.Value == r.Value
	case *Float:
		r := right.(*Float)
		return l.Value == r.Value
	case *String:
		r := right.(*String)
		return l.Value == r.Value
	case *Boolean:
		r := right.(*Boolean)
		return l.Value == r.Value
	case *Null:
		return true
	}
	return false
}

func isEqualLoose(left, right Object) bool {
	// If identical, they are loosely equal too
	if isEqualIdentical(left, right) {
		return true
	}

	// Handle numeric comparisons
	if isNumeric(left) && isNumeric(right) {
		return toFloat(left) == toFloat(right)
	}

	// Fallback to string comparisons for mixed/others
	return toString(left) == toString(right)
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

func newError(format string, a ...interface{}) *Error {
	return &Error{Message: fmt.Sprintf(format, a...)}
}

func isError(obj Object) bool {
	if obj != nil {
		return obj.Type() == ERROR_OBJ
	}
	return false
}

func applyFunction(fn *Function, args []Object, out io.Writer) Object {
	extendedEnv := NewEnclosedEnvironment(fn.Env)
	for i, param := range fn.Parameters {
		if i < len(args) {
			extendedEnv.Set(param.Var.Value, args[i])
		} else if param.DefaultValue != nil {
			val := Evaluate(param.DefaultValue, fn.Env, out)
			if isError(val) {
				return val
			}
			extendedEnv.Set(param.Var.Value, val)
		} else {
			extendedEnv.Set(param.Var.Value, NULL)
		}
	}
	evaluated := Evaluate(fn.Body, extendedEnv, out)
	return unwrapReturnValue(evaluated)
}

func unwrapReturnValue(obj Object) Object {
	if returnValue, ok := obj.(*ReturnValue); ok {
		return returnValue.Value
	}
	return obj
}

func interpolateString(s string, env *Environment, out io.Writer) Object {
	var result []byte
	i := 0
	n := len(s)
	for i < n {
		if s[i] == '\\' && i+1 < n {
			if s[i+1] == '$' || s[i+1] == '{' || s[i+1] == '}' {
				result = append(result, s[i+1])
				i += 2
				continue
			}
		}
		if i+1 < n && s[i] == '{' && s[i+1] == '$' {
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
				exprStr := s[start:end]
				val := parseAndEvalExpression(exprStr, env, out)
				result = append(result, []byte(toString(val))...)
				i = end + 1
				continue
			}
		} else if s[i] == '$' {
			start := i
			i++
			for i < n && (isAlphaNumeric(s[i]) || s[i] == '_') {
				i++
			}
			if i+2 < n && s[i] == '-' && s[i+1] == '>' && (isLetter(s[i+2]) || s[i+2] == '_') {
				i += 2
				for i < n && (isAlphaNumeric(s[i]) || s[i] == '_') {
					i++
				}
			}
			exprStr := s[start:i]
			val := parseAndEvalExpression(exprStr, env, out)
			result = append(result, []byte(toString(val))...)
			continue
		}
		result = append(result, s[i])
		i++
	}
	return &String{Value: unescapeString(string(result))}
}

func parseAndEvalExpression(exprStr string, env *Environment, out io.Writer) Object {
	fullCode := "<?php " + exprStr + "; ?>"
	l := lexer.New(fullCode)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 || len(prog.Statements) == 0 {
		return &String{Value: exprStr}
	}
	stmt, ok := prog.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		return &String{Value: exprStr}
	}
	val := Evaluate(stmt.Expression, env, out)
	if isError(val) {
		return val
	}
	return val
}

func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func getTargetValue(target ast.Expression, env *Environment, out io.Writer) Object {
	switch t := target.(type) {
	case *ast.Variable:
		val, ok := env.Get(t.Value)
		if !ok {
			return NewInteger(0)
		}
		return val
	case *ast.IndexExpression:
		leftVal := Evaluate(t.Left, env, out)
		if isError(leftVal) {
			return leftVal
		}
		arr, ok := leftVal.(*Array)
		if !ok {
			return newError("cannot read index of non-array: %s", leftVal.Type())
		}
		if t.Index == nil {
			return newError("cannot use [] for reading")
		}
		idxVal := Evaluate(t.Index, env, out)
		if isError(idxVal) {
			return idxVal
		}
		for _, pair := range arr.Pairs {
			if keysEqual(pair.Key, idxVal) {
				return pair.Value
			}
		}
		return NULL
	case *ast.PropertyExpression:
		objVal := Evaluate(t.Object, env, out)
		if isError(objVal) {
			return objVal
		}
		instance, ok := objVal.(*ObjectInstance)
		if !ok {
			return newError("cannot read property of non-object: %s", t.Object.String())
		}
		val, ok := instance.Fields[t.Property.Value]
		if !ok {
			return NewInteger(0)
		}
		return val
	default:
		return newError("invalid expression target: %s", target.String())
	}
}

func setTargetValue(target ast.Expression, val Object, env *Environment, out io.Writer) Object {
	switch t := target.(type) {
	case *ast.Variable:
		env.Set(t.Value, val)
		return val
	case *ast.IndexExpression:
		var arr *Array
		if v, ok := t.Left.(*ast.Variable); ok {
			leftVal, exists := env.Get(v.Value)
			if !exists {
				arr = &Array{Pairs: []*ArrayPair{}}
				env.Set(v.Value, arr)
			} else {
				var ok bool
				arr, ok = leftVal.(*Array)
				if !ok {
					return newError("cannot set index on non-array: %s", leftVal.Type())
				}
			}
		} else {
			leftVal := Evaluate(t.Left, env, out)
			if isError(leftVal) {
				return leftVal
			}
			var ok bool
			arr, ok = leftVal.(*Array)
			if !ok {
				return newError("cannot set index on non-array: %s", leftVal.Type())
			}
		}

		if t.Index == nil {
			var nextIdx int64 = 0
			for _, pair := range arr.Pairs {
				if intKey, ok := pair.Key.(*Integer); ok {
					if intKey.Value >= nextIdx {
						nextIdx = intKey.Value + 1
					}
				}
			}
			arr.Pairs = append(arr.Pairs, &ArrayPair{Key: NewInteger(nextIdx), Value: val})
			return val
		}

		idxVal := Evaluate(t.Index, env, out)
		if isError(idxVal) {
			return idxVal
		}

		found := false
		for _, pair := range arr.Pairs {
			if keysEqual(pair.Key, idxVal) {
				pair.Value = val
				found = true
				break
			}
		}
		if !found {
			arr.Pairs = append(arr.Pairs, &ArrayPair{Key: idxVal, Value: val})
		}
		return val

	case *ast.PropertyExpression:
		objVal := Evaluate(t.Object, env, out)
		if isError(objVal) {
			return objVal
		}
		instance, ok := objVal.(*ObjectInstance)
		if !ok {
			return newError("cannot set property of non-object: %s", t.Object.String())
		}
		instance.Fields[t.Property.Value] = val
		return val
	default:
		return newError("invalid assignment target: %s", target.String())
	}
}

func keysEqual(k1, k2 Object) bool {
	if k1.Type() != k2.Type() {
		return false
	}
	switch o1 := k1.(type) {
	case *Integer:
		return o1.Value == k2.(*Integer).Value
	case *String:
		return o1.Value == k2.(*String).Value
	case *Boolean:
		return o1.Value == k2.(*Boolean).Value
	default:
		return k1.Inspect() == k2.Inspect()
	}
}
