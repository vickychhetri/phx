package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"phx/internal/ast"
	"phx/internal/lexer"
	"phx/internal/parser"
	"phx/internal/vm"
	"strconv"
)

var EmbeddedMainPHP string
var EmbeddedFiles = map[string]string{}

func main() {
	if EmbeddedMainPHP != "" {
		vm.VirtualFilesystem = EmbeddedFiles
		l := lexer.New(EmbeddedMainPHP)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			fmt.Println("Parser Errors encountered:")
			for _, errStr := range p.Errors() {
				fmt.Printf("  - %s\n", errStr)
			}
			os.Exit(1)
		}
		env := vm.NewEnvironment()
		result := vm.Evaluate(program, env, os.Stdout)
		if errObj, ok := result.(*vm.Error); ok {
			fmt.Fprintf(os.Stderr, "Runtime Error: %s\n", errObj.Message)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "build":
		buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
		outputFlag := buildCmd.String("o", "", "Output binary name")
		buildCmd.Parse(os.Args[2:])
		args := buildCmd.Args()
		if len(args) < 1 {
			fmt.Println("Usage: phx build -o <output> <main-file>")
			os.Exit(1)
		}
		mainFilePath := args[0]
		outputBinary := *outputFlag
		if outputBinary == "" {
			outputBinary = "app"
		}
		runBuild(mainFilePath, outputBinary)

	case "parse":
		parseCmd := flag.NewFlagSet("parse", flag.ExitOnError)
		jsonFlag := parseCmd.Bool("json", false, "Output AST as JSON")
		stringFlag := parseCmd.Bool("string", false, "Output reconstructed PHP string")

		parseCmd.Parse(os.Args[2:])
		args := parseCmd.Args()
		if len(args) < 1 {
			fmt.Println("Error: Missing file path")
			fmt.Println("Usage: phx parse [--json] [--string] <file>")
			os.Exit(1)
		}

		filePath := args[0]
		runParse(filePath, *jsonFlag, *stringFlag)

	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Usage: phx run <file>")
			os.Exit(1)
		}
		filePath := os.Args[2]
		runInterpreter(filePath)

	case "help", "-h", "--help":
		printUsage()

	default:
		fmt.Printf("Unknown command: %q\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("PHX - PHP Native Runtime")
	fmt.Println("Usage:")
	fmt.Println("  phx parse [--json] [--string] <file>  - Parse file and display AST")
	fmt.Println("  phx run <file>                       - Run PHP script (Stub/Interpreter)")
	fmt.Println("  phx build -o <output> <main-file>    - Compile and build PHP script to machine binary")
	fmt.Println("  phx help                             - Show this help message")
}

func runParse(filePath string, outputJSON bool, outputString bool) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file %q: %v\n", filePath, err)
		os.Exit(1)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Parser Errors encountered:")
		for _, errStr := range p.Errors() {
			fmt.Printf("  - %s\n", errStr)
		}
		os.Exit(1)
	}

	if outputJSON {
		jsonBytes, err := json.MarshalIndent(astToMap(program), "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling to JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonBytes))
	} else if outputString {
		fmt.Println(program.String())
	} else {
		printNode(program, "")
	}
}

func runInterpreter(filePath string) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file %q: %v\n", filePath, err)
		os.Exit(1)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Parser Errors encountered:")
		for _, errStr := range p.Errors() {
			fmt.Printf("  - %s\n", errStr)
		}
		os.Exit(1)
	}

	env := vm.NewEnvironment()
	result := vm.Evaluate(program, env, os.Stdout)
	if errObj, ok := result.(*vm.Error); ok {
		fmt.Fprintf(os.Stderr, "Runtime Error: %s\n", errObj.Message)
		os.Exit(1)
	}
}

func printNode(node ast.Node, indent string) {
	if node == nil {
		fmt.Printf("%sNil\n", indent)
		return
	}
	switch n := node.(type) {
	case *ast.Program:
		fmt.Println("Program")
		for _, stmt := range n.Statements {
			printNode(stmt, indent+"  ")
		}
	case *ast.InlineHTMLStatement:
		fmt.Printf("%sInlineHTML: %q\n", indent, n.Content)
	case *ast.EchoStatement:
		fmt.Printf("%sEchoStatement\n", indent)
		for _, expr := range n.Expressions {
			printNode(expr, indent+"  ")
		}
	case *ast.ExpressionStatement:
		fmt.Printf("%sExpressionStatement\n", indent)
		printNode(n.Expression, indent+"  ")
	case *ast.BlockStatement:
		fmt.Printf("%sBlockStatement\n", indent)
		for _, stmt := range n.Statements {
			printNode(stmt, indent+"  ")
		}
	case *ast.IfStatement:
		fmt.Printf("%sIfStatement\n", indent)
		fmt.Printf("%s  Condition:\n", indent)
		printNode(n.Condition, indent+"    ")
		fmt.Printf("%s  Consequence:\n", indent)
		printNode(n.Consequence, indent+"    ")
		if n.Alternative != nil {
			fmt.Printf("%s  Alternative:\n", indent)
			printNode(n.Alternative, indent+"      ")
		}
	case *ast.AssignExpression:
		fmt.Printf("%sAssignExpression (=)\n", indent)
		fmt.Printf("%s  Left:\n", indent)
		printNode(n.Left, indent+"    ")
		fmt.Printf("%s  Value:\n", indent)
		printNode(n.Value, indent+"    ")
	case *ast.Variable:
		fmt.Printf("%sVariable: %s\n", indent, n.Value)
	case *ast.Identifier:
		fmt.Printf("%sIdentifier: %s\n", indent, n.Value)
	case *ast.IntegerLiteral:
		fmt.Printf("%sIntegerLiteral: %d\n", indent, n.Value)
	case *ast.FloatLiteral:
		fmt.Printf("%sFloatLiteral: %f\n", indent, n.Value)
	case *ast.StringLiteral:
		fmt.Printf("%sStringLiteral: %q\n", indent, n.Value)
	case *ast.BooleanLiteral:
		fmt.Printf("%sBooleanLiteral: %t\n", indent, n.Value)
	case *ast.NullLiteral:
		fmt.Printf("%sNullLiteral\n", indent)
	case *ast.InfixExpression:
		fmt.Printf("%sInfixExpression (%s)\n", indent, n.Operator)
		fmt.Printf("%s  Left:\n", indent)
		printNode(n.Left, indent+"    ")
		fmt.Printf("%s  Right:\n", indent)
		printNode(n.Right, indent+"    ")
	default:
		fmt.Printf("%sUnknownNode(%T)\n", indent, node)
	}
}

func astToMap(node ast.Node) interface{} {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.Program:
		stmts := make([]interface{}, len(n.Statements))
		for i, s := range n.Statements {
			stmts[i] = astToMap(s)
		}
		return map[string]interface{}{
			"type":       "Program",
			"statements": stmts,
		}
	case *ast.InlineHTMLStatement:
		return map[string]interface{}{
			"type":    "InlineHTMLStatement",
			"content": n.Content,
		}
	case *ast.EchoStatement:
		exprs := make([]interface{}, len(n.Expressions))
		for i, e := range n.Expressions {
			exprs[i] = astToMap(e)
		}
		return map[string]interface{}{
			"type":        "EchoStatement",
			"expressions": exprs,
		}
	case *ast.ExpressionStatement:
		return map[string]interface{}{
			"type":       "ExpressionStatement",
			"expression": astToMap(n.Expression),
		}
	case *ast.BlockStatement:
		stmts := make([]interface{}, len(n.Statements))
		for i, s := range n.Statements {
			stmts[i] = astToMap(s)
		}
		return map[string]interface{}{
			"type":       "BlockStatement",
			"statements": stmts,
		}
	case *ast.IfStatement:
		result := map[string]interface{}{
			"type":        "IfStatement",
			"condition":   astToMap(n.Condition),
			"consequence": astToMap(n.Consequence),
		}
		if n.Alternative != nil {
			result["alternative"] = astToMap(n.Alternative)
		}
		return result
	case *ast.AssignExpression:
		return map[string]interface{}{
			"type":  "AssignExpression",
			"left":  astToMap(n.Left),
			"value": astToMap(n.Value),
		}
	case *ast.Variable:
		return map[string]interface{}{
			"type":  "Variable",
			"value": n.Value,
		}
	case *ast.Identifier:
		return map[string]interface{}{
			"type":  "Identifier",
			"value": n.Value,
		}
	case *ast.IntegerLiteral:
		return map[string]interface{}{
			"type":  "IntegerLiteral",
			"value": n.Value,
		}
	case *ast.FloatLiteral:
		return map[string]interface{}{
			"type":  "FloatLiteral",
			"value": n.Value,
		}
	case *ast.StringLiteral:
		return map[string]interface{}{
			"type":  "StringLiteral",
			"value": n.Value,
		}
	case *ast.BooleanLiteral:
		return map[string]interface{}{
			"type":  "BooleanLiteral",
			"value": n.Value,
		}
	case *ast.NullLiteral:
		return map[string]interface{}{
			"type": "NullLiteral",
		}
	case *ast.InfixExpression:
		return map[string]interface{}{
			"type":     "InfixExpression",
			"operator": n.Operator,
			"left":     astToMap(n.Left),
			"right":    astToMap(n.Right),
		}
	default:
		return fmt.Sprintf("UnknownNode(%T)", node)
	}
}

func runBuild(mainFilePath string, outputBinary string) {
	absMainPath, err := filepath.Abs(mainFilePath)
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		os.Exit(1)
	}

	mainDir := filepath.Dir(absMainPath)

	mainContent, err := ioutil.ReadFile(absMainPath)
	if err != nil {
		fmt.Printf("Error reading main file: %v\n", err)
		os.Exit(1)
	}

	embeddedFiles := make(map[string]string)
	err = filepath.Walk(mainDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".php" {
			rel, err := filepath.Rel(mainDir, path)
			if err != nil {
				return err
			}
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			embeddedFiles[rel] = string(content)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		os.Exit(1)
	}

	// Create temp folder
	tempDir := filepath.Join(".", "_phx_build_temp")
	err = os.MkdirAll(tempDir, 0755)
	if err != nil {
		fmt.Printf("Error creating temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	var buf bytes.Buffer
	buf.WriteString("package main\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"fmt\"\n")
	buf.WriteString("\t\"os\"\n")
	buf.WriteString("\t\"phx/internal/lexer\"\n")
	buf.WriteString("\t\"phx/internal/parser\"\n")
	buf.WriteString("\t\"phx/internal/vm\"\n")
	buf.WriteString(")\n\n")

	// Embed main script
	buf.WriteString(fmt.Sprintf("var EmbeddedMainPHP = %s\n\n", strconv.Quote(string(mainContent))))

	// Embed other files
	buf.WriteString("var EmbeddedFiles = map[string]string{\n")
	for relPath, content := range embeddedFiles {
		buf.WriteString(fmt.Sprintf("\t%s: %s,\n", strconv.Quote(relPath), strconv.Quote(content)))
	}
	buf.WriteString("}\n\n")

	// Main function
	buf.WriteString(`func main() {
	vm.VirtualFilesystem = EmbeddedFiles
	l := lexer.New(EmbeddedMainPHP)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		fmt.Println("Parser Errors encountered:")
		for _, errStr := range p.Errors() {
			fmt.Printf("  - %s\n", errStr)
		}
		os.Exit(1)
	}
	env := vm.NewEnvironment()
	result := vm.Evaluate(program, env, os.Stdout)
	if errObj, ok := result.(*vm.Error); ok {
		fmt.Fprintf(os.Stderr, "Runtime Error: %s\n", errObj.Message)
		os.Exit(1)
	}
}
`)

	tempGoFile := filepath.Join(tempDir, "main.go")
	err = ioutil.WriteFile(tempGoFile, buf.Bytes(), 0644)
	if err != nil {
		fmt.Printf("Error writing temporary Go file: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.Command("go", "build", "-o", outputBinary, tempGoFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error running go build: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully built standalone binary: %s\n", outputBinary)
}
