package vm

import (
	"bytes"
	"fmt"
	"io"
	"phx/internal/ast"
)

type ObjectType string

const (
	INTEGER_OBJ = "INTEGER"
	FLOAT_OBJ   = "FLOAT"
	BOOLEAN_OBJ = "BOOLEAN"
	STRING_OBJ  = "STRING"
	NULL_OBJ    = "NULL"
	ERROR_OBJ   = "ERROR"
	FUNCTION_OBJ = "FUNCTION"
	BUILTIN_OBJ  = "BUILTIN"
	RETURN_VALUE_OBJ = "RETURN_VALUE"
	CLASS_OBJ   = "CLASS"
	INSTANCE_OBJ = "INSTANCE"
	BREAK_OBJ   = "BREAK"
	CONTINUE_OBJ = "CONTINUE"
	THREAD_OBJ  = "THREAD"
	CHANNEL_OBJ = "CHANNEL"
	ARRAY_OBJ   = "ARRAY"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

const (
	intCacheMin = -100
	intCacheMax = 50000
)

var intCache [intCacheMax - intCacheMin + 1]*Integer

func init() {
	for i := intCacheMin; i <= intCacheMax; i++ {
		intCache[i-intCacheMin] = &Integer{Value: int64(i)}
	}
}

func NewInteger(val int64) *Integer {
	if val >= intCacheMin && val <= intCacheMax {
		return intCache[val-intCacheMin]
	}
	return &Integer{Value: val}
}

type Float struct {
	Value float64
}

func (f *Float) Type() ObjectType { return FLOAT_OBJ }
func (f *Float) Inspect() string  { return fmt.Sprintf("%g", f.Value) }

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "NULL" }

type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return "ERROR: " + e.Message }

type Function struct {
	Parameters []*ast.Parameter
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string  { return "function" }

type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

type Class struct {
	Name    string
	Methods map[string]*Function
}

func (c *Class) Type() ObjectType { return CLASS_OBJ }
func (c *Class) Inspect() string  { return "class " + c.Name }

type ObjectInstance struct {
	Class  *Class
	Fields map[string]Object
}

func (oi *ObjectInstance) Type() ObjectType { return INSTANCE_OBJ }
func (oi *ObjectInstance) Inspect() string  { return "object of class " + oi.Class.Name }

func NewObjectInstance(class *Class) *ObjectInstance {
	return &ObjectInstance{
		Class:  class,
		Fields: make(map[string]Object),
	}
}

type Break struct{}
func (b *Break) Type() ObjectType { return BREAK_OBJ }
func (b *Break) Inspect() string  { return "break" }

type Continue struct{}
func (c *Continue) Type() ObjectType { return CONTINUE_OBJ }
func (c *Continue) Inspect() string  { return "continue" }

type Thread struct {
	Done chan struct{}
	Val  Object
	Err  Object
}
func (t *Thread) Type() ObjectType { return THREAD_OBJ }
func (t *Thread) Inspect() string  { return "<Thread>" }

type Channel struct {
	Ch chan Object
}
func (c *Channel) Type() ObjectType { return CHANNEL_OBJ }
func (c *Channel) Inspect() string  { return "<Channel>" }

type BuiltinFunction func(args []Object, env *Environment, out io.Writer) Object

type Builtin struct {
	Fn BuiltinFunction
}
func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "<builtin function>" }

type ArrayPair struct {
	Key   Object
	Value Object
}

type Array struct {
	Pairs []*ArrayPair
}
func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	var out bytes.Buffer
	out.WriteString("Array ( ")
	for _, pair := range a.Pairs {
		out.WriteString(fmt.Sprintf("[%s] => %s ", pair.Key.Inspect(), pair.Value.Inspect()))
	}
	out.WriteString(")")
	return out.String()
}
