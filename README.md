# PHX – A Native PHP Compiler & Runtime

**PHX** is a high-performance transpilation engine, compiler, and runtime for a PHP‑compatible programming language, written from the ground up in Go. Rather than relying on traditional slow tree-walking or virtual machine interpretation, PHX parses PHP source files, translates the Abstract Syntax Tree (AST) directly into highly optimized Go code, and compiles it via the native Go toolchain (`go build`) to produce self-contained, high-performance machine executables.

PHX supports PHP-style syntax (`$`, `->`, arrays, classes, loops, etc.) alongside native Go concurrency primitives (`spawn`, `channels`) and standard enterprise modules (MySQL, PostgreSQL, MongoDB, File I/O, Try-Catch Exception handling).

---

## Table of Contents

1. [Features](#features)  
2. [Repository Structure](#repository-structure)  
3. [Getting Started](#getting-started)  
4. [Performance Benchmark](#performance-benchmark)  
5. [Advanced Features & Optimizations](#advanced-features--optimizations)  
6. [Multi-File Static Linking](#multi-file-static-linking)
7. [Running Tests & Building](#running-tests--building)  
8. [Documentation & Tutorials](#documentation--tutorials)
9. [License](#license)

---

## Features

- **PHP‑style syntax**: Full support for variables, classes, objects, multidimensional arrays, closures, and control flow structures (including `switch-case` and ternary expressions).
- **AST-to-Go Transpiler**: Compiles PHP code into native Go source code, entirely bypassing VM overhead.
- **Type Specialization Engine**: Statically traces integer-based mathematical expressions and loops to generate raw machine integer operations, bypassing allocations.
- **Advanced Concurrency**: Native green thread support via `spawn()` (maps to goroutines) and safe synchronization via `channel()`, `send()`, and `receive()`.
- **Comprehensive Standard Library**: Built-in support for string operations, math, file handling, and database drivers (MySQL, Postgres, MongoDB).
- **Error Handling**: Full `try`, `catch`, and `throw` support built on top of native Go panic-recovery.
- **Ahead-of-Time (AOT) Inlining**: Fully links included files (`include`/`require`) at compile time.

---

## Repository Structure

```
phx/
├─ bin/                     # Generated CLI binaries
├─ cmd/phx/main.go          # CLI entry point (run, build, parse)
├─ docs/
│   └─ internals.html       # Book-style technical internals documentation
├─ tutorial/
│   └─ index.html           # Minimalist black & white tutorial book
├─ project_demo/            # Sample multi-file services project
│   ├─ services/            # LogService, MathService, UserService
│   └─ main.php             # Main project entry point
├─ examples/                # Example scripts (concurrency, databases, files)
├─ internal/
│   ├─ ast/                 # AST node definitions
│   ├─ lexer/               # Lexical scanner (HTML & PHP mixed modes)
│   ├─ parser/              # Pratt top-down operator precedence parser
│   ├─ token/               # Token definitions
│   ├─ compiler/            # AST-to-Go transpiler and runtime library header
│   └─ vm/                  # Development AST-interpreter
├─ go.mod
└─ README.md (this file)
```

---

## Getting Started

### 1. Build the compiler CLI:
```bash
go build -o bin/phx ./cmd/phx
```

### 2. Compile and run a program dynamically:
```bash
bin/phx run examples/01_hello_world.php
```

### 3. Compile a script into a standalone machine binary:
```bash
bin/phx build -o bin/myprog examples/02_variables_operators.php
./bin/myprog
```

---

## Performance Benchmark

We measured CPU performance using a multi-threaded prime-number summation algorithm running over **50,000,000** iterations.

| Runtime / Engine | Execution Mode | Parallel Workers | Execution Time | Speedup vs Native PHP |
| :--- | :--- | :--- | :--- | :--- |
| **Native PHP 8.x** | Interpreted / VM | 1 (Single-Thread) | **~148.49 sec** | 1.0x (Baseline) |
| **PHX Native Compiler** | Go Machine Code | 4 (Goroutines) | **~12.40 sec** | **~12.0x Faster** |

---

## Advanced Features & Optimizations

### 1. Static Type Specialization
Before code generation, the compiler runs a tree analysis (`isIntExpr`). If mathematical statements, loop boundaries, or comparison expressions involve only integer values and variables, the generator emits raw Go `int64` operations (e.g. `v_i++`). This eliminates Zend-style boxing, heap allocations, and type validation overhead inside tight loops.

### 2. Value Capture Closure IIFEs
PHP handles anonymous closure variables by value using the `use` keyword. To prevent standard Go loop variable closure references from pointing to final iteration values, PHX compiles closure bindings inside Immediately Invoked Function Expressions (IIFEs) in Go:
```go
func(v_x Val) Val {
    return Val{Type: 8, Func: func(args ...Val) Val {
        return v_x
    }}
}(v_x)
```

### 3. Thread Synchronization (`ThreadObject`)
Calling `spawn(function)` runs a background thread. To retrieve the result, PHX implements a `ThreadObject` with a blocking `join()` method that resolves values across internal Go channels cleanly.

---

## Multi-File Static Linking

PHX compiles multiple source files ahead-of-time (AOT) using the `include` and `require` keywords:
- File paths are resolved and parsed at compile-time.
- The transpiler inlines the target file's AST directly into the parent code.
- Hoisted local variables from all included modules are collected and declared at the top of the Go program block to guarantee Go compiler compatibility.

For example, see our modular demo project:
```bash
bin/phx run project_demo/main.php
```

---

## Running Tests & Building

Run all unit tests across the codebase:
```bash
go test -v ./...
```

---

## Documentation & Tutorials

- **Detailed Technical Guide**: Read about the Pratt parser, the lexer state-machine, and the type specialization engine inside [docs/internals.html](file:///home/vicky/languages/phx/docs/internals.html).
- **Language Tutorial**: Learn to write code in PHX (from basics to threads) with the minimalist black & white tutorial at [tutorial/index.html](file:///home/vicky/languages/phx/tutorial/index.html).

---

## Package Management (alias: pkg)

PHX uses Go modules for dependency management. Use the `pkg` command to manage your project's dependencies:

```bash
bin/phx pkg init <project-name>

bin/phx pkg add <github-username>/<repo-name>

bin/phx pkg update

```

### Example

```bash
# Create a new project
bin/phx pkg init my-web-service

```

---
## License

PHX is released under the **MIT License**. See the `LICENSE` file for details.
