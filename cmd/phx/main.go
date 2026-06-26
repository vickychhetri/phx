package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"phx/internal/ast"
	"phx/internal/compiler"
	"phx/internal/lexer"
	"phx/internal/parser"
	"phx/internal/pm"
	"phx/internal/vm"
)

var EmbeddedMainPHP string
var EmbeddedFiles = map[string]string{}

const tempGoMod = `module temp

go 1.25.10

require (
	github.com/go-sql-driver/mysql v1.10.0
	github.com/lib/pq v1.12.3
)

require filippo.io/edwards25519 v1.2.0 // indirect
`

const tempGoSum = `filippo.io/edwards25519 v1.2.0 h1:crnVqOiS4jqYleHd9vaKZ+HKtHfllngJIiOpNpoJsjo=
filippo.io/edwards25519 v1.2.0/go.mod h1:xzAOLCNug/yB62zG1bQ8uziwrIqIuxhctzJT18Q77mc=
github.com/go-sql-driver/mysql v1.10.0 h1:Q+1LV8DkHJvSYAdR83XzuhDaTykuDx0l6fkXxoWCWfw=
github.com/go-sql-driver/mysql v1.10.0/go.mod h1:M+cqaI7+xxXGG9swrdeUIoPG3Y3KCkF0pZej+SK+nWk=
github.com/lib/pq v1.12.3 h1:tTWxr2YLKwIvK90ZXEw8GP7UFHtcbTtty8zsI+YjrfQ=
github.com/lib/pq v1.12.3/go.mod h1:/p+8NSbOcwzAEI7wiMXFlgydTwcgTr3OSKMsD2BitpA=
`

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
		if result != nil && result.Type() == "EXCEPTION" {
			fmt.Fprintf(os.Stderr, "Uncaught Exception: %s\n", result.Inspect())
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

	case "package", "pkg":
		if len(os.Args) < 3 {
			printPackageUsage()
			os.Exit(1)
		}
		subcommand := os.Args[2]
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current working directory: %v\n", err)
			os.Exit(1)
		}
		switch subcommand {
		case "init":
			yes := false
			if len(os.Args) > 3 && (os.Args[3] == "-y" || os.Args[3] == "--yes") {
				yes = true
			}
			err = pm.Init(cwd, yes)
		case "add":
			if len(os.Args) < 4 {
				fmt.Println("Usage: phx package add <package-name> [constraint]")
				os.Exit(1)
			}
			pkgName := os.Args[3]
			constraint := ""
			if len(os.Args) > 4 {
				constraint = os.Args[4]
			}
			err = pm.Add(cwd, pkgName, constraint)
		case "install":
			err = pm.Install(cwd, false)
		case "update":
			err = pm.Update(cwd)
		case "publish":
			err = pm.Publish(cwd)
		case "search":
			query := ""
			if len(os.Args) > 3 {
				query = os.Args[3]
			}
			err = pm.Search(query)
		default:
			fmt.Printf("Unknown package subcommand: %q\n", subcommand)
			printPackageUsage()
			os.Exit(1)
		}
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

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
	fmt.Println("  phx package <subcommand> [args]      - Manage dependencies (alias: pkg)")
	fmt.Println("  phx help                             - Show this help message")
}

func printPackageUsage() {
	fmt.Println("Usage: phx package <subcommand> [args] (alias: pkg)")
	fmt.Println("Subcommands:")
	fmt.Println("  init [-y|--yes]                  - Initialize a new phx.json manifest")
	fmt.Println("  add <package> [constraint]       - Add a dependency and install it")
	fmt.Println("  install                          - Install dependencies from phx.lock or phx.json")
	fmt.Println("  update                           - Update dependencies to latest versions")
	fmt.Println("  publish                          - Publish the package to the local registry")
	fmt.Println("  search <query>                   - Search for packages in the registry")
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

	// Compile PHP AST to Go code
	comp := compiler.New()
	goCode, err := comp.Compile(program, filePath)
	if err != nil {
		fmt.Printf("Compilation Error: %v\n", err)
		os.Exit(1)
	}

	// Create temporary directory for building
	tempDir, err := ioutil.TempDir("", "phx_run_")
	if err != nil {
		fmt.Printf("Error creating temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	tempModFile := filepath.Join(tempDir, "go.mod")
	err = ioutil.WriteFile(tempModFile, []byte(tempGoMod), 0644)
	if err != nil {
		fmt.Printf("Error writing temporary go.mod: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	tempSumFile := filepath.Join(tempDir, "go.sum")
	err = ioutil.WriteFile(tempSumFile, []byte(tempGoSum), 0644)
	if err != nil {
		fmt.Printf("Error writing temporary go.sum: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	tempGoFile := filepath.Join(tempDir, "main.go")
	err = ioutil.WriteFile(tempGoFile, []byte(goCode), 0644)
	if err != nil {
		fmt.Printf("Error writing temporary Go file: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	tempBin := filepath.Join(tempDir, "app")
	buildCmd := exec.Command("go", "build", "-o", tempBin, "main.go")
	buildCmd.Dir = tempDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	err = buildCmd.Run()
	if err != nil {
		fmt.Printf("Error compiling binary: %v\n", err)
		_ = ioutil.WriteFile("scratch_main.go", []byte(goCode), 0644)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	// Execute binary
	runCmd := exec.Command(tempBin)
	runCmd.Stdin = os.Stdin
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	err = runCmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.RemoveAll(tempDir)
			os.Exit(exitErr.ExitCode())
		}
		fmt.Printf("Error running binary: %v\n", err)
		os.RemoveAll(tempDir)
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

	mainContent, err := ioutil.ReadFile(absMainPath)
	if err != nil {
		fmt.Printf("Error reading main file: %v\n", err)
		os.Exit(1)
	}

	l := lexer.New(string(mainContent))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Parser Errors encountered:")
		for _, errStr := range p.Errors() {
			fmt.Printf("  - %s\n", errStr)
		}
		os.Exit(1)
	}

	// Compile to Go
	comp := compiler.New()
	goCode, err := comp.Compile(program, absMainPath)
	if err != nil {
		fmt.Printf("Compilation Error: %v\n", err)
		os.Exit(1)
	}

	// Create temp folder
	tempDir, err := ioutil.TempDir("", "phx_build_")
	if err != nil {
		fmt.Printf("Error creating temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	tempModFile := filepath.Join(tempDir, "go.mod")
	err = ioutil.WriteFile(tempModFile, []byte(tempGoMod), 0644)
	if err != nil {
		fmt.Printf("Error writing temporary go.mod: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	tempSumFile := filepath.Join(tempDir, "go.sum")
	err = ioutil.WriteFile(tempSumFile, []byte(tempGoSum), 0644)
	if err != nil {
		fmt.Printf("Error writing temporary go.sum: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	tempGoFile := filepath.Join(tempDir, "main.go")
	err = ioutil.WriteFile(tempGoFile, []byte(goCode), 0644)
	if err != nil {
		fmt.Printf("Error writing temporary Go file: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	absOutputBinary, err := filepath.Abs(outputBinary)
	if err != nil {
		fmt.Printf("Error getting absolute path for output: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	cmd := exec.Command("go", "build", "-o", absOutputBinary, "main.go")
	cmd.Dir = tempDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error running go build: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	fmt.Printf("Successfully built standalone binary: %s\n", outputBinary)
}
