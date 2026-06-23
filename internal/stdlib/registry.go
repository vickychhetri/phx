package stdlib

import (
	"io"
	"phx/internal/vm"
)

// PackageLoader is a function that registers builtins into an environment.
type PackageLoader func(env *vm.Environment, out io.Writer)

// registry maps fully qualified PHX package paths to their loader functions.
var registry = map[string]PackageLoader{}

// Register adds a package to the global registry.
func Register(name string, loader PackageLoader) {
	registry[name] = loader
}

// Load registers all builtins for the given package name into env.
// Returns true if the package was found.
func Load(name string, env *vm.Environment, out io.Writer) bool {
	loader, ok := registry[name]
	if !ok {
		return false
	}
	loader(env, out)
	return true
}

// IsPackage reports whether name is a known PHX standard library package.
func IsPackage(name string) bool {
	_, ok := registry[name]
	return ok
}
