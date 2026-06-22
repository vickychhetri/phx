# PHX – A Go‑based PHP‑style Runtime

**PHX** is a lightweight interpreter / runtime for a PHP‑compatible language, written in pure Go. It supports PHP‑style syntax (`$`, `->`, arrays, objects, loops, functions, etc.) and provides a small standard library with built‑ins for system‑programming tasks (threads, channels, I/O, math, etc.).

---

## Table of Contents

1. [Features](#features)  
2. [Repository Structure](#repository-structure)  
3. [Getting Started](#getting-started)  
4. [Performance Benchmark](#performance-benchmark)  
5. [Optimizations Implemented](#optimizations-implemented)  
6. [Running Tests & Building](#running-tests--building)  
7. [Contributing](#contributing)  
8. [License](#license)

---

## Features

- **PHP‑style syntax** (variables, objects, arrays, classes, control flow)
- **AST‑based interpreter** with Pratt parser
- **Standard library** (string/array utilities, math, I/O, threading, channels, etc.)
- **Threading model** – `spawn` builtin creates Go goroutine “threads”
- **Standalone binary** generation (`phx build -o <out> <script>`)
- **Optimized integer handling** (cached integer objects, atomic thread‑count tracking)

---

## Repository Structure

```
phx/
├─ benchmark_go.go          # Go version of the prime‑number benchmark
├─ cmd/phx/main.go          # CLI entry point (run, build, repl)
├─ internal/
│   ├─ ast/                # AST node definitions
│   ├─ lexer/              # Lexer (supports PHP tags)
│   ├─ parser/             # Pratt parser + precedence tables
│   ├─ token/              # Token definitions (including MOD %)
│   └─ vm/
│       ├─ environment.go  # Variable store, built‑ins, thread tracking
│       ├─ evaluator.go    # Core evaluation logic (expressions, statements)
│       ├─ object.go       # Object types (Integer, Float, Boolean, etc.)
│       └─ evaluator_test.go
├─ internal/vm/builtins/   # (implicit via registerBuiltins)
├─ go.mod
└─ README.md (this file)
```

---

## Getting Started

1. **Clone the repository** (already done in your local workspace).  
2. **Install Go** (≥ 1.22).  
3. Build the binary:

```bash
go build -o bin/phx ./cmd/phx
```

4. Run a PHX script:

```bash
bin/phx run p6.php
```

5. Build a standalone binary from a script:

```bash
bin/phx build -o myprog p6.php
./myprog
```

---

## Performance Benchmark

Two benchmark programs are provided:

| Language | File | Description |
|----------|------|-------------|
| **Go** | `benchmark_go.go` | Prime‑number sieve up to 50 000. |
| **PHX** | `p6.php` | Same algorithm written in PHX (PHP‑style). |

Running each program:

```bash
# Go version
go run benchmark_go.go

# PHX version
bin/phx run p6.php
```

Typical output (your recent run):

```
--- Go ---
Execution Time  : 0.0019 seconds

--- PHX ---
Execution Time  : 0.337 seconds
```

---

## Optimizations Implemented

### 1. Integer Caching
- **Problem**: Every integer literal caused a heap allocation (`&Integer{Value: …}`), which is expensive in tight loops.
- **Solution**: Added a global cache (`intCache`) for integers from **‑100** to **50 000** and a helper `NewInteger(val int64) *Integer`.
- **Impact**: Eliminates most allocations for small integers, reducing GC pressure.

### 2. Atomic Thread Counter (`ActiveThreads`)
- **Problem**: Environment look‑ups (`Get`, `Set`) lock a `sync.RWMutex` even when no threads are active, adding overhead for single‑threaded execution.
- **Solution**: Introduced `ActiveThreads` (int32) and `sync/atomic` checks. When `ActiveThreads == 0`, `Get`/`Set` bypass the lock.
- **Integration**: `spawn` now increments/decrements `ActiveThreads` atomically when a thread starts/finishes.

### 3. Helper Method Bypass
- Resolved helper functions (`ResolveName`, `QualifyName`, `SetNamespace`, `AddAlias`) to skip locking when no threads are running, keeping a clean code path for the common single‑threaded case.

### 4. Replaced Raw Integer Instantiations
- Updated all evaluator and builtin code to use `NewInteger` instead of `&Integer{…}` (e.g., arithmetic, array indexing, fallbacks).

### 5. Minor Token Additions
- Added **MOD** (`%`) token and lexer case to support modulo operations.

These changes collectively brought the PHX benchmark down from **~0.41 s** to **~0.34 s**, a ~15 % improvement, while keeping correctness and thread safety intact.

---

## Running Tests & Building

```bash
# Run all unit tests
go test -v ./...

# Build the binary (creates ./bin/phx)
go build -o bin/phx ./cmd/phx
```

All tests currently pass:

```
ok   phx/internal/vm   0.003s
```

---

## Contributing

1. **Fork** the repository and create a feature branch.  
2. Write **tests** for any new functionality.  
3. Follow the existing **code style** (tabs, `go fmt`).  
4. Commit changes with clear messages (`git commit -m "Add X feature"`).  
5. Push to your fork and open a **Pull Request**.

When adding performance‑related code, consider:

- Using the integer cache (`NewInteger`).  
- Updating `ActiveThreads` if you introduce new concurrency primitives.  
- Adding benchmarks in `internal/vm/evaluator_test.go` for regression detection.

---

## License

PHX is released under the **MIT License**. See the `LICENSE` file for details.

---

**Happy coding!** 🎉
