package vm

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Environment struct {
	mu               sync.RWMutex
	store            map[string]Object
	outer            *Environment
	currentNamespace string
	aliases          map[string]string
}

func NewEnvironment() *Environment {
	env := &Environment{
		store:            make(map[string]Object),
		outer:            nil,
		currentNamespace: "",
		aliases:          make(map[string]string),
	}
	registerBuiltins(env)
	return env
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	return &Environment{
		store:            make(map[string]Object),
		outer:            outer,
		currentNamespace: "",
		aliases:          make(map[string]string),
	}
}

var ActiveThreads int32

func (e *Environment) Get(name string) (Object, bool) {
	if atomic.LoadInt32(&ActiveThreads) > 0 {
		e.mu.RLock()
		obj, ok := e.store[name]
		e.mu.RUnlock()
		
		if !ok && e.outer != nil {
			obj, ok = e.outer.Get(name)
		}
		return obj, ok
	}

	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	if atomic.LoadInt32(&ActiveThreads) > 0 {
		e.mu.Lock()
		e.store[name] = val
		e.mu.Unlock()
		return val
	}
	e.store[name] = val
	return val
}

func (e *Environment) ResolveName(name string) string {
	if atomic.LoadInt32(&ActiveThreads) > 0 {
		e.mu.RLock()
		defer e.mu.RUnlock()
		return e.resolveNameUnlocked(name)
	}
	return e.resolveNameUnlocked(name)
}

func (e *Environment) resolveNameUnlocked(name string) string {
	// First check local aliases
	if e.aliases != nil {
		if alias, ok := e.aliases[name]; ok {
			return alias
		}
	}
	// Delegate to outer if local alias not found
	if e.outer != nil {
		return e.outer.ResolveName(name)
	}
	// Resolve based on currentNamespace
	if e.currentNamespace != "" {
		combined := e.currentNamespace + "\\" + name
		// Check global store
		if _, ok := e.Get(combined); ok {
			return combined
		}
	}
	return name
}

func (e *Environment) QualifyName(name string) string {
	if atomic.LoadInt32(&ActiveThreads) > 0 {
		e.mu.RLock()
		defer e.mu.RUnlock()
		return e.qualifyNameUnlocked(name)
	}
	return e.qualifyNameUnlocked(name)
}

func (e *Environment) qualifyNameUnlocked(name string) string {
	if e.currentNamespace != "" && !strings.Contains(name, "\\") {
		return e.currentNamespace + "\\" + name
	}
	if e.outer != nil {
		return e.outer.QualifyName(name)
	}
	return name
}

func (e *Environment) SetNamespace(ns string) {
	if atomic.LoadInt32(&ActiveThreads) > 0 {
		e.mu.Lock()
		e.currentNamespace = ns
		e.mu.Unlock()
		return
	}
	e.currentNamespace = ns
}

func (e *Environment) AddAlias(alias, name string) {
	if atomic.LoadInt32(&ActiveThreads) > 0 {
		e.mu.Lock()
		e.aliases[alias] = name
		e.mu.Unlock()
		return
	}
	e.aliases[alias] = name
}

func registerBuiltins(env *Environment) {
	env.Set("spawn", &Builtin{
		Fn: func(args []Object, callerEnv *Environment, out io.Writer) Object {
			if len(args) == 0 {
				return newError("spawn expects at least 1 argument")
			}
			fnObj := args[0]
			callable, isFn := fnObj.(*Function)
			if !isFn {
				return newError("spawn first argument must be a function")
			}
			threadArgs := args[1:]
			t := &Thread{
				Done: make(chan struct{}),
			}
			atomic.AddInt32(&ActiveThreads, 1)
			go func() {
				defer func() {
					close(t.Done)
					atomic.AddInt32(&ActiveThreads, -1)
				}()
				extendedEnv := NewEnclosedEnvironment(callable.Env)
				for i, param := range callable.Parameters {
					if i < len(threadArgs) {
						extendedEnv.Set(param.Var.Value, threadArgs[i])
					} else if param.DefaultValue != nil {
						val := Evaluate(param.DefaultValue, callable.Env, out)
						if isError(val) {
							t.Err = val
							return
						}
						extendedEnv.Set(param.Var.Value, val)
					} else {
						extendedEnv.Set(param.Var.Value, NULL)
					}
				}
				res := Evaluate(callable.Body, extendedEnv, out)
				if isError(res) {
					t.Err = res
				} else {
					t.Val = unwrapReturnValue(res)
				}
			}()
			return t
		},
	})

	env.Set("sleep", &Builtin{
		Fn: func(args []Object, callerEnv *Environment, out io.Writer) Object {
			if len(args) == 0 {
				return newError("sleep expects 1 argument")
			}
			secVal, ok := args[0].(*Integer)
			if !ok {
				return newError("sleep argument must be an integer")
			}
			time.Sleep(time.Duration(secVal.Value) * time.Second)
			return NULL
		},
	})

	env.Set("count", &Builtin{
		Fn: func(args []Object, callerEnv *Environment, out io.Writer) Object {
			if len(args) == 0 {
				return newError("count expects 1 argument")
			}
			switch obj := args[0].(type) {
			case *Array:
				return NewInteger(int64(len(obj.Pairs)))
			case *String:
				return NewInteger(int64(len(obj.Value)))
			default:
				return NewInteger(1)
			}
		},
	})

	env.Set("print_r", &Builtin{
		Fn: func(args []Object, callerEnv *Environment, out io.Writer) Object {
			if len(args) == 0 {
				return newError("print_r expects at least 1 argument")
			}
			out.Write([]byte(args[0].Inspect() + "\n"))
			return TRUE
		},
	})

	env.Set("strlen", &Builtin{
		Fn: func(args []Object, callerEnv *Environment, out io.Writer) Object {
			if len(args) == 0 {
				return newError("strlen expects 1 argument")
			}
			return NewInteger(int64(len(args[0].Inspect())))
		},
	})

	env.Set("strpos", &Builtin{
		Fn: func(args []Object, callerEnv *Environment, out io.Writer) Object {
			if len(args) < 2 {
				return newError("strpos expects 2 arguments")
			}
			haystack := args[0].Inspect()
			needle := args[1].Inspect()
			idx := strings.Index(haystack, needle)
			if idx == -1 {
				return FALSE
			}
			return NewInteger(int64(idx))
		},
	})

	env.Set("substr", &Builtin{
		Fn: func(args []Object, callerEnv *Environment, out io.Writer) Object {
			if len(args) < 2 {
				return newError("substr expects 2 or 3 arguments")
			}
			str := args[0].Inspect()
			startVal, ok1 := args[1].(*Integer)
			if !ok1 {
				return newError("substr start index must be integer")
			}
			start := int(startVal.Value)
			if start < 0 {
				start = len(str) + start
			}
			if start < 0 {
				start = 0
			}
			if start > len(str) {
				return FALSE
			}

			length := len(str) - start
			if len(args) >= 3 {
				lenVal, ok2 := args[2].(*Integer)
				if !ok2 {
					return newError("substr length must be integer")
				}
				l := int(lenVal.Value)
				if l < 0 {
					length = len(str) - start + l
				} else {
					length = l
				}
			}
			if start+length > len(str) {
				length = len(str) - start
			}
			if length <= 0 {
				return &String{Value: ""}
			}
			return &String{Value: str[start : start+length]}
		},
	})

	env.Set("readline", &Builtin{
		Fn: func(args []Object, callerEnv *Environment, out io.Writer) Object {
			if len(args) > 0 {
				out.Write([]byte(args[0].Inspect()))
			}
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				return &String{Value: scanner.Text()}
			}
			return FALSE
		},
	})

	env.Set("intval", &Builtin{
		Fn: func(args []Object, callerEnv *Environment, out io.Writer) Object {
			if len(args) == 0 {
				return &Integer{Value: 0}
			}
			switch obj := args[0].(type) {
			case *Integer:
				return obj
			case *Float:
				return &Integer{Value: int64(obj.Value)}
			case *String:
				i, _ := strconv.ParseInt(obj.Value, 10, 64)
				return &Integer{Value: i}
			case *Boolean:
				if obj.Value {
					return &Integer{Value: 1}
				}
				return &Integer{Value: 0}
			default:
				return &Integer{Value: 0}
			}
		},
	})

	env.Set("floatval", &Builtin{
		Fn: func(args []Object, callerEnv *Environment, out io.Writer) Object {
			if len(args) == 0 {
				return &Float{Value: 0.0}
			}
			switch obj := args[0].(type) {
			case *Integer:
				return &Float{Value: float64(obj.Value)}
			case *Float:
				return obj
			case *String:
				f, _ := strconv.ParseFloat(obj.Value, 64)
				return &Float{Value: f}
			case *Boolean:
				if obj.Value {
					return &Float{Value: 1.0}
				}
				return &Float{Value: 0.0}
			default:
				return &Float{Value: 0.0}
			}
		},
	})

	env.Set("strval", &Builtin{
		Fn: func(args []Object, callerEnv *Environment, out io.Writer) Object {
			if len(args) == 0 {
				return &String{Value: ""}
			}
			return &String{Value: args[0].Inspect()}
		},
	})

	env.Set("abs", &Builtin{
		Fn: func(args []Object, callerEnv *Environment, out io.Writer) Object {
			if len(args) == 0 {
				return newError("abs expects 1 argument")
			}
			switch val := args[0].(type) {
			case *Integer:
				if val.Value < 0 {
					return &Integer{Value: -val.Value}
				}
				return val
			case *Float:
				if val.Value < 0 {
					return &Float{Value: -val.Value}
				}
				return val
			default:
				return newError("abs expects a number")
			}
		},
	})

	env.Set("microtime", &Builtin{
		Fn: func(args []Object, callerEnv *Environment, out io.Writer) Object {
			getFloat := false
			if len(args) > 0 {
				if b, ok := args[0].(*Boolean); ok {
					getFloat = b.Value
				}
			}
			now := time.Now()
			if getFloat {
				return &Float{Value: float64(now.UnixNano()) / 1e9}
			}
			sec := now.Unix()
			msec := float64(now.UnixNano()%1e9) / 1e9
			return &String{Value: fmt.Sprintf("%f %d", msec, sec)}
		},
	})

	env.Set("min", &Builtin{
		Fn: func(args []Object, callerEnv *Environment, out io.Writer) Object {
			if len(args) < 2 {
				return newError("min expects at least 2 arguments")
			}
			minVal := args[0]
			for _, arg := range args[1:] {
				if isLessThan(arg, minVal) {
					minVal = arg
				}
			}
			return minVal
		},
	})

	env.Set("max", &Builtin{
		Fn: func(args []Object, callerEnv *Environment, out io.Writer) Object {
			if len(args) < 2 {
				return newError("max expects at least 2 arguments")
			}
			maxVal := args[0]
			for _, arg := range args[1:] {
				if isGreaterThan(arg, maxVal) {
					maxVal = arg
				}
			}
			return maxVal
		},
	})
}

func isLessThan(a, b Object) bool {
	switch va := a.(type) {
	case *Integer:
		if vb, ok := b.(*Integer); ok {
			return va.Value < vb.Value
		}
	case *Float:
		if vb, ok := b.(*Float); ok {
			return va.Value < vb.Value
		}
	}
	return a.Inspect() < b.Inspect()
}

func isGreaterThan(a, b Object) bool {
	switch va := a.(type) {
	case *Integer:
		if vb, ok := b.(*Integer); ok {
			return va.Value > vb.Value
		}
	case *Float:
		if vb, ok := b.(*Float); ok {
			return va.Value > vb.Value
		}
	}
	return a.Inspect() > b.Inspect()
}
