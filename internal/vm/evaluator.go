package vm

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
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

	case *ast.ThrowStatement:
		val := Evaluate(n.Expression, env, out)
		if isError(val) {
			return val
		}
		return &ExceptionObject{Value: val}

	case *ast.TryCatchStatement:
		result := Evaluate(n.TryBlock, env, out)
		if result != nil && result.Type() == EXCEPTION_OBJ {
			exceptionObj := result.(*ExceptionObject).Value
			resolvedCatchClass := env.ResolveName(n.CatchClass)
			matches := false

			if inst, ok := exceptionObj.(*ObjectInstance); ok {
				if inst.Class.Name == resolvedCatchClass || strings.HasSuffix(inst.Class.Name, "\\"+resolvedCatchClass) {
					matches = true
				}
			}

			// Fallback match for Exception/PHX\Exception/PHX\Error
			if !matches && (resolvedCatchClass == "Exception" || resolvedCatchClass == "PHX\\Exception" || resolvedCatchClass == "PHX\\Error") {
				matches = true
			}

			if matches {
				catchEnv := NewEnclosedEnvironment(env)
				catchEnv.Set(n.CatchVar, exceptionObj)
				return Evaluate(n.CatchBlock, catchEnv, out)
			}
			return result
		}
		return result

	case *ast.WhileStatement:
		whileLoop: for {
			condition := Evaluate(n.Condition, env, out)
			if isError(condition) {
				return condition
			}
			if !isTruthy(condition) {
				break
			}
			result := Evaluate(n.Body, env, out)
			if result != nil {
				switch result.(type) {
				case *Error, *ReturnValue, *ExceptionObject:
					return result
				case *Break:
					break whileLoop
				case *Continue:
					continue whileLoop
				}
			}
		}
		return NULL

	case *ast.DoWhileStatement:
		doWhileLoop: for {
			result := Evaluate(n.Body, env, out)
			if result != nil {
				switch result.(type) {
				case *Error, *ReturnValue, *ExceptionObject:
					return result
				case *Break:
					break doWhileLoop
				case *Continue:
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

	case *ast.ForeachStatement:
		arrObj := Evaluate(n.Expression, env, out)
		if isError(arrObj) {
			return arrObj
		}
		arr, ok := arrObj.(*Array)
		if !ok {
			return newError("foreach expression is not an array: %s", arrObj.Type())
		}

		foreachLoop: for _, pair := range arr.Pairs {
			loopEnv := NewEnclosedEnvironment(env)
			if n.Key != nil {
				loopEnv.Set(n.Key.Value, pair.Key)
			}
			loopEnv.Set(n.Value.Value, pair.Value)

			result := Evaluate(n.Body, loopEnv, out)
			if result != nil {
				switch result.(type) {
				case *Error, *ReturnValue, *ExceptionObject:
					return result
				case *Break:
					break foreachLoop
				case *Continue:
					continue foreachLoop
				}
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
		forLoop: for {
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
				switch result.(type) {
				case *Error, *ReturnValue, *ExceptionObject:
					return result
				case *Break:
					break forLoop
				case *Continue:
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

		if n.Token.Type == token.ADD_ASSIGN || n.Token.Type == token.SUB_ASSIGN || n.Token.Type == token.CONCAT_ASSIGN {
			currVal := getTargetValue(n.Left, env, out)
			if isError(currVal) {
				return currVal
			}
			op := "+"
			if n.Token.Type == token.SUB_ASSIGN {
				op = "-"
			} else if n.Token.Type == token.CONCAT_ASSIGN {
				op = "."
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
		if n.Operator == "::" {
			var classObj Object
			leftIdent, leftOk := n.Left.(*ast.Identifier)
			if leftOk && leftIdent.Value == "self" {
				thisObj, ok := env.Get("$this")
				if ok {
					if inst, ok := thisObj.(*ObjectInstance); ok {
						classObj = inst.Class
					}
				}
			}
			if classObj == nil {
				leftName := ""
				if leftOk {
					leftName = leftIdent.Value
				} else if leftVar, ok := n.Left.(*ast.Variable); ok {
					leftName = leftVar.Value
				}
				if leftName != "" {
					resolvedClass := env.ResolveName(leftName)
					if cObj, ok := env.Get(resolvedClass); ok {
						classObj = cObj
					}
				}
			}
			if classObj != nil {
				if cls, ok := classObj.(*Class); ok {
					rightIdent, rightOk := n.Right.(*ast.Identifier)
					if rightOk {
						if val, ok := cls.Constants[rightIdent.Value]; ok {
							return val
						}
						return newError("undefined class constant: %s::%s", cls.Name, rightIdent.Value)
					}
				}
			}
			return newError("could not resolve class for :: operator")
		}

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
		closureEnv := env
		if len(n.UseVars) > 0 {
			closureEnv = NewEnclosedEnvironment(env)
			for _, v := range n.UseVars {
				if val, ok := env.Get(v.Value); ok {
					closureEnv.Set(v.Value, val)
				} else {
					closureEnv.Set(v.Value, NULL)
				}
			}
		}
		return &Function{Parameters: n.Parameters, Body: n.Body, Env: closureEnv}

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
		constants := make(map[string]Object)
		for _, cst := range n.Constants {
			constants[cst.Name] = Evaluate(cst.Value, env, out)
		}
		qualifiedName := env.QualifyName(n.Name.Value)
		classObj := &Class{Name: qualifiedName, Methods: methods, Constants: constants}
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
		if resolvedClass == "PHX\\File" || resolvedClass == "File" {
			return &FileObject{}
		}
		if resolvedClass == "PHX\\Database" || resolvedClass == "PHX\\DB" || resolvedClass == "Database" || resolvedClass == "DB" {
			return &DatabaseObject{}
		}
		if resolvedClass == "PHX\\MySQL" || resolvedClass == "MySQL" {
			return &MySQLObject{}
		}
		if resolvedClass == "PHX\\PostgreSQL" || resolvedClass == "Postgres" || resolvedClass == "PostgreSQL" || resolvedClass == "PHX\\Postgres" {
			return &PostgresObject{}
		}
		if resolvedClass == "PHX\\MongoDB" || resolvedClass == "Mongo" || resolvedClass == "MongoDB" || resolvedClass == "PHX\\Mongo" {
			return &MongoObject{collections: make(map[string][]Object)}
		}
		if resolvedClass == "PHX\\Exception" || resolvedClass == "Exception" || resolvedClass == "PHX\\Error" || resolvedClass == "Error" {
			msg := ""
			var code int64 = 0
			if len(n.Arguments) > 0 {
				argVal := Evaluate(n.Arguments[0], env, out)
				if isError(argVal) {
					return argVal
				}
				msg = argVal.Inspect()
				// strip quotes from inspect if it's a string representation
				if strings.HasPrefix(msg, `"`) && strings.HasSuffix(msg, `"`) {
					msg = msg[1 : len(msg)-1]
				}
			}
			if len(n.Arguments) > 1 {
				codeVal := Evaluate(n.Arguments[1], env, out)
				if isError(codeVal) {
					return codeVal
				}
				if intVal, ok := codeVal.(*Integer); ok {
					code = intVal.Value
				}
			}
			return &PHXExceptionObject{Message: msg, Code: code}
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

		if fileObj, ok := objVal.(*FileObject); ok {
			methodName := n.Method.Value
			args := []Object{}
			for _, arg := range n.Arguments {
				val := Evaluate(arg, env, out)
				if isError(val) {
					return val
				}
				args = append(args, val)
			}

			switch methodName {
			case "open":
				if len(args) < 2 {
					return newError("File::open expects 2 arguments: path and mode")
				}
				path := args[0].Inspect()
				mode := args[1].Inspect()

				// Strip quotes if they were string inspected
				if strings.HasPrefix(path, `"`) && strings.HasSuffix(path, `"`) {
					path = path[1 : len(path)-1]
				}
				if strings.HasPrefix(mode, `"`) && strings.HasSuffix(mode, `"`) {
					mode = mode[1 : len(mode)-1]
				}

				var flag int
				if strings.Contains(mode, "w") {
					flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
				} else if strings.Contains(mode, "a") {
					flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
				} else {
					flag = os.O_RDONLY
				}

				f, err := os.OpenFile(path, flag, 0666)
				if err != nil {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "failed to open file: " + err.Error()}}
				}
				fileObj.file = f
				return TRUE

			case "read":
				if fileObj.file == nil {
					return newError("file is not open")
				}
				if len(args) < 1 {
					return newError("File::read expects 1 argument: length")
				}
				length, ok := args[0].(*Integer)
				if !ok {
					return newError("File::read first argument must be an integer")
				}
				buf := make([]byte, length.Value)
				nBytes, err := fileObj.file.Read(buf)
				if err != nil && err != io.EOF {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "failed to read file: " + err.Error()}}
				}
				return &String{Value: string(buf[:nBytes])}

			case "readLine":
				if fileObj.file == nil {
					return newError("file is not open")
				}
				reader := bufio.NewReader(fileObj.file)
				line, err := reader.ReadString('\n')
				if err != nil && err != io.EOF {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "failed to read line: " + err.Error()}}
				}
				if line == "" && err == io.EOF {
					return FALSE
				}
				return &String{Value: line}

			case "write":
				if fileObj.file == nil {
					return newError("file is not open")
				}
				if len(args) < 1 {
					return newError("File::write expects 1 argument: content")
				}
				content := args[0].Inspect()
				if strings.HasPrefix(content, `"`) && strings.HasSuffix(content, `"`) {
					content = content[1 : len(content)-1]
				}
				nBytes, err := fileObj.file.WriteString(content)
				if err != nil {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "failed to write file: " + err.Error()}}
				}
				return &Integer{Value: int64(nBytes)}

			case "close":
				if fileObj.file != nil {
					fileObj.file.Close()
					fileObj.file = nil
				}
				return TRUE

			case "exists":
				if len(args) < 1 {
					return newError("File::exists expects 1 argument: path")
				}
				path := args[0].Inspect()
				if strings.HasPrefix(path, `"`) && strings.HasSuffix(path, `"`) {
					path = path[1 : len(path)-1]
				}
				_, err := os.Stat(path)
				if err == nil {
					return TRUE
				}
				return FALSE

			case "delete":
				if len(args) < 1 {
					return newError("File::delete expects 1 argument: path")
				}
				path := args[0].Inspect()
				if strings.HasPrefix(path, `"`) && strings.HasSuffix(path, `"`) {
					path = path[1 : len(path)-1]
				}
				err := os.Remove(path)
				if err != nil {
					return FALSE
				}
				return TRUE

			default:
				return newError("undefined method: %s on File", methodName)
			}
		}

		if dbObj, ok := objVal.(*DatabaseObject); ok {
			methodName := n.Method.Value
			args := []Object{}
			for _, arg := range n.Arguments {
				val := Evaluate(arg, env, out)
				if isError(val) {
					return val
				}
				args = append(args, val)
			}

			switch methodName {
			case "open":
				if len(args) < 1 {
					return newError("Database::open expects 1 argument: dbPath")
				}
				dbPath := args[0].Inspect()
				if strings.HasPrefix(dbPath, `"`) && strings.HasSuffix(dbPath, `"`) {
					dbPath = dbPath[1 : len(dbPath)-1]
				}
				dbObj.path = dbPath
				dbObj.data = make(map[string][]map[string]Object)

				if _, err := os.Stat(dbPath); err == nil {
					content, err := os.ReadFile(dbPath)
					if err == nil {
						dbObj.loadFromJSON(content)
					}
				}
				return TRUE

			case "exec":
				if dbObj.data == nil {
					return newError("database is not open")
				}
				if len(args) < 1 {
					return newError("Database::exec expects 1 argument: sqlQuery")
				}
				query := args[0].Inspect()
				if strings.HasPrefix(query, `"`) && strings.HasSuffix(query, `"`) {
					query = query[1 : len(query)-1]
				}
				err := dbObj.executeExec(query)
				if err != nil {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "DB Exec Error: " + err.Error()}}
				}
				return TRUE

			case "query":
				if dbObj.data == nil {
					return newError("database is not open")
				}
				if len(args) < 1 {
					return newError("Database::query expects 1 argument: sqlQuery")
				}
				query := args[0].Inspect()
				if strings.HasPrefix(query, `"`) && strings.HasSuffix(query, `"`) {
					query = query[1 : len(query)-1]
				}
				res, err := dbObj.executeQuery(query)
				if err != nil {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "DB Query Error: " + err.Error()}}
				}
				return res

			case "close":
				dbObj.data = nil
				dbObj.path = ""
				return TRUE

			default:
				return newError("undefined method: %s on Database", methodName)
			}
		}

		if myObj, ok := objVal.(*MySQLObject); ok {
			methodName := n.Method.Value
			args := []Object{}
			for _, arg := range n.Arguments {
				val := Evaluate(arg, env, out)
				if isError(val) {
					return val
				}
				args = append(args, val)
			}

			switch methodName {
			case "connect":
				if len(args) < 4 {
					return newError("MySQL::connect expects 4 arguments: host, user, password, db")
				}
				host := strings.Trim(args[0].Inspect(), `"` + `'`)
				user := strings.Trim(args[1].Inspect(), `"` + `'`)
				password := strings.Trim(args[2].Inspect(), `"` + `'`)
				db := strings.Trim(args[3].Inspect(), `"` + `'`)
				err := myObj.connect(host, user, password, db)
				if err != nil {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "MySQL Connection Error: " + err.Error()}}
				}
				fmt.Printf("[MySQL] Connected to %s@%s db: %s\n", user, host, db)
				return TRUE

			case "exec":
				if !myObj.connected {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "MySQL Error: Not connected"}}
				}
				if len(args) < 1 {
					return newError("MySQL::exec expects 1 argument: sqlQuery")
				}
				query := args[0].Inspect()
				query = strings.Trim(query, `"` + `'`)
				err := myObj.executeExec(query)
				if err != nil {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "MySQL Exec Error: " + err.Error()}}
				}
				return TRUE

			case "query":
				if !myObj.connected {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "MySQL Error: Not connected"}}
				}
				if len(args) < 1 {
					return newError("MySQL::query expects 1 argument: sqlQuery")
				}
				query := args[0].Inspect()
				query = strings.Trim(query, `"` + `'`)
				res, err := myObj.executeQuery(query)
				if err != nil {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "MySQL Query Error: " + err.Error()}}
				}
				return res

			case "close":
				myObj.close()
				fmt.Println("[MySQL] Connection closed")
				return TRUE

			default:
				return newError("undefined method: %s on MySQL", methodName)
			}
		}

		if pgObj, ok := objVal.(*PostgresObject); ok {
			methodName := n.Method.Value
			args := []Object{}
			for _, arg := range n.Arguments {
				val := Evaluate(arg, env, out)
				if isError(val) {
					return val
				}
				args = append(args, val)
			}

			switch methodName {
			case "connect":
				if len(args) < 4 {
					return newError("PostgreSQL::connect expects 4 arguments: host, user, password, db, [port]")
				}
				host := strings.Trim(args[0].Inspect(), `"` + `'`)
				user := strings.Trim(args[1].Inspect(), `"` + `'`)
				password := strings.Trim(args[2].Inspect(), `"` + `'`)
				db := strings.Trim(args[3].Inspect(), `"` + `'`)

				var port int64 = 5432
				if len(args) > 4 {
					if intVal, ok := args[4].(*Integer); ok {
						port = intVal.Value
					}
				}
				err := pgObj.connect(host, user, password, db, port)
				if err != nil {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "PostgreSQL Connection Error: " + err.Error()}}
				}
				fmt.Printf("[PostgreSQL] Connected to %s@%s:%d db: %s\n", user, host, port, db)
				return TRUE

			case "exec":
				if !pgObj.connected {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "PostgreSQL Error: Not connected"}}
				}
				if len(args) < 1 {
					return newError("PostgreSQL::exec expects 1 argument: sqlQuery")
				}
				query := args[0].Inspect()
				query = strings.Trim(query, `"` + `'`)
				err := pgObj.executeExec(query)
				if err != nil {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "PostgreSQL Exec Error: " + err.Error()}}
				}
				return TRUE

			case "query":
				if !pgObj.connected {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "PostgreSQL Error: Not connected"}}
				}
				if len(args) < 1 {
					return newError("PostgreSQL::query expects 1 argument: sqlQuery")
				}
				query := args[0].Inspect()
				query = strings.Trim(query, `"` + `'`)
				res, err := pgObj.executeQuery(query)
				if err != nil {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "PostgreSQL Query Error: " + err.Error()}}
				}
				return res

			case "close":
				pgObj.close()
				fmt.Println("[PostgreSQL] Connection closed")
				return TRUE

			default:
				return newError("undefined method: %s on PostgreSQL", methodName)
			}
		}

		if mgObj, ok := objVal.(*MongoObject); ok {
			methodName := n.Method.Value
			args := []Object{}
			for _, arg := range n.Arguments {
				val := Evaluate(arg, env, out)
				if isError(val) {
					return val
				}
				args = append(args, val)
			}

			switch methodName {
			case "connect":
				if len(args) < 1 {
					return newError("MongoDB::connect expects 1 argument: uri")
				}
				mgObj.uri = args[0].Inspect()
				mgObj.uri = strings.Trim(mgObj.uri, `"` + `'`)
				mgObj.connected = true
				fmt.Printf("[MongoDB] Connected to %s\n", mgObj.uri)
				return TRUE

			case "selectDatabase":
				if !mgObj.connected {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "MongoDB Error: Not connected"}}
				}
				if len(args) < 1 {
					return newError("MongoDB::selectDatabase expects 1 argument: dbName")
				}
				mgObj.dbName = args[0].Inspect()
				mgObj.dbName = strings.Trim(mgObj.dbName, `"` + `'`)
				return TRUE

			case "selectCollection":
				if !mgObj.connected {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "MongoDB Error: Not connected"}}
				}
				if len(args) < 1 {
					return newError("MongoDB::selectCollection expects 1 argument: collectionName")
				}
				mgObj.collName = args[0].Inspect()
				mgObj.collName = strings.Trim(mgObj.collName, `"` + `'`)
				return TRUE

			case "insertOne":
				if !mgObj.connected {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "MongoDB Error: Not connected"}}
				}
				if len(args) < 1 {
					return newError("MongoDB::insertOne expects 1 argument: document")
				}
				doc := args[0]
				if mgObj.collections == nil {
					mgObj.collections = make(map[string][]Object)
				}
				mgObj.collections[mgObj.collName] = append(mgObj.collections[mgObj.collName], doc)
				return TRUE

			case "find":
				if !mgObj.connected {
					return &ExceptionObject{Value: &PHXExceptionObject{Message: "MongoDB Error: Not connected"}}
				}
				filter := Object(nil)
				if len(args) > 0 {
					filter = args[0]
				}

				docs, ok := mgObj.collections[mgObj.collName]
				if !ok {
					return &Array{Pairs: []*ArrayPair{}}
				}

				results := []*ArrayPair{}
				idx := int64(0)

				for _, doc := range docs {
					matches := true
					if filterArr, isArr := filter.(*Array); isArr && filter != nil {
						for _, filterPair := range filterArr.Pairs {
							filterKey := filterPair.Key.Inspect()
							filterKey = strings.Trim(filterKey, `"` + `'` + `[` + `]`)
							filterVal := filterPair.Value.Inspect()
							filterVal = strings.Trim(filterVal, `"` + `'`)

							if docArr, docIsArr := doc.(*Array); docIsArr {
								fieldMatched := false
								for _, docPair := range docArr.Pairs {
									docKey := docPair.Key.Inspect()
									docKey = strings.Trim(docKey, `"` + `'` + `[` + `]`)
									if docKey == filterKey {
										docVal := docPair.Value.Inspect()
										docVal = strings.Trim(docVal, `"` + `'`)
										if docVal == filterVal {
											fieldMatched = true
											break
										}
									}
								}
								if !fieldMatched {
									matches = false
									break
								}
							} else {
								matches = false
								break
							}
						}
					}

					if matches {
						results = append(results, &ArrayPair{
							Key:   &Integer{Value: idx},
							Value: doc,
						})
						idx++
					}
				}

				return &Array{Pairs: results}

			case "close":
				mgObj.connected = false
				fmt.Println("[MongoDB] Connection closed")
				return TRUE

			default:
				return newError("undefined method: %s on MongoDB", methodName)
			}
		}

		if exObj, ok := objVal.(*PHXExceptionObject); ok {
			methodName := n.Method.Value
			switch methodName {
			case "getMessage":
				return &String{Value: exObj.Message}
			case "getCode":
				return &Integer{Value: exObj.Code}
			default:
				return newError("undefined method: %s on Exception", methodName)
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
			switch r := result.(type) {
			case *ReturnValue:
				return r.Value
			case *Error, *ExceptionObject, *Break, *Continue:
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
			switch result.(type) {
			case *ReturnValue, *Error, *ExceptionObject, *Break, *Continue:
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
	case "@":
		return right
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
	case "&&", "and":
		return nativeBoolToBooleanObject(isTruthy(left) && isTruthy(right))
	case "||", "or":
		return nativeBoolToBooleanObject(isTruthy(left) || isTruthy(right))
	case "??":
		if left == NULL {
			return right
		}
		return left
	case "&":
		return &Integer{Value: toIntegerVal(left) & toIntegerVal(right)}
	case "|":
		return &Integer{Value: toIntegerVal(left) | toIntegerVal(right)}
	}

	// Numerical expressions
	if isNumeric(left) && isNumeric(right) {
		return evalNumericInfixExpression(operator, left, right)
	}

	// Relational string comparisons
	lStr, lOk := left.(*String)
	rStr, rOk := right.(*String)
	if lOk && rOk {
		lVal := lStr.Value
		rVal := rStr.Value
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
	_, leftIsFloat := left.(*Float)
	_, rightIsFloat := right.(*Float)
	isFloatOp := leftIsFloat || rightIsFloat

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
	if obj == nil {
		return false
	}
	switch obj.(type) {
	case *Integer, *Float:
		return true
	default:
		return false
	}
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

func toIntegerVal(obj Object) int64 {
	switch o := obj.(type) {
	case *Integer:
		return o.Value
	case *Float:
		return int64(o.Value)
	case *Boolean:
		if o.Value {
			return 1
		}
		return 0
	case *String:
		var i int64
		fmt.Sscanf(o.Value, "%d", &i)
		return i
	default:
		return 0
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
	if obj == nil {
		return false
	}
	switch obj.(type) {
	case *Error, *ExceptionObject:
		return true
	default:
		return false
	}
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
