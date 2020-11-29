package eval

import (
	"fmt"
	"reflect"
)

// An entry in a stack trace
type StackEntry struct {
	Function string
	Line     int
}

type Object interface {
	Type() string
}

type Exception interface {
	Object
	Msg() string
	AddStackEntry(StackEntry)
	GetStackTrace() string
}

func Equal(o1, o2 Object) bool {
	return reflect.DeepEqual(o1, o2)
}

type StringObject struct {
	val string
}

func (t *StringObject) Type() string {
	return "str"
}

func (t *StringObject) String() string {
	return fmt.Sprintf("%s(%v)", t.Type(), t.val)
}

type IntObject struct {
	val int
}

func (t *IntObject) Type() string {
	return "int"
}

func (t *IntObject) String() string {
	return fmt.Sprintf("%s(%v)", t.Type(), t.val)
}

type ExnObject struct {
	val   string
	stack []StackEntry
}

func (t *ExnObject) Type() string {
	return "exception"
}

func (t *ExnObject) String() string {
	return fmt.Sprintf("%s(%v)", t.Type(), t.val)
}

func (t *ExnObject) Msg() string {
	return t.val
}

func (t *ExnObject) AddStackEntry(entry StackEntry) {
	t.stack = append(t.stack, entry)
}

func (t *ExnObject) GetStackTrace() string {
	res := ""
	for i := len(t.stack) - 1; i >= 0; i-- {
		e := t.stack[i]
		res += fmt.Sprintf("  unknown:%d - %s", e.Line, e.Function)
		if i > 0 {
			res += "\n"
		}
	}
	return res
}

type ExitObject struct {
	ExnObject
	ExitCode int
}

func (t *ExitObject) Type() string {
	return "exit"
}

type UnitObject struct {
}

func (t *UnitObject) Type() string {
	return "()"
}

func (t *UnitObject) String() string {
	return "()"
}

func add(o1, o2 Object) Object {
	i1, ok := o1.(*IntObject)
	if !ok {
		panic("trying to add non-integer")
	}
	i2, ok := o2.(*IntObject)
	if !ok {
		panic("trying to add non-integer")
	}
	return IntVal(i1.val + i2.val)
}

func sub(o1, o2 Object) Object {
	i1, ok := o1.(*IntObject)
	if !ok {
		panic("trying to sub non-integer")
	}
	i2, ok := o2.(*IntObject)
	if !ok {
		panic("trying to sub non-integer")
	}
	return IntVal(i1.val - i2.val)
}

func mult(o1, o2 Object) Object {
	i1, ok := o1.(*IntObject)
	if !ok {
		panic("trying to mult non-integer")
	}
	i2, ok := o2.(*IntObject)
	if !ok {
		panic("trying to mult non-integer")
	}
	return IntVal(i1.val * i2.val)
}

func div(o1, o2 Object) Object {
	i1, ok := o1.(*IntObject)
	if !ok {
		panic("trying to div non-integer")
	}
	i2, ok := o2.(*IntObject)
	if !ok {
		panic("trying to div non-integer")
	}
	return IntVal(i1.val / i2.val)
}

func neg(o Object) Object {
	i, ok := o.(*IntObject)
	if !ok {
		panic("trying to negate non-integer")
	}
	return IntVal(-i.val)
}

func IntVal(n int) *IntObject {
	return &IntObject{val: n}
}

func StrVal(s string) *StringObject {
	return &StringObject{val: s}
}

func ExitVal(n int, cause string, line int) *ExitObject {

	return &ExitObject{
		*ExnVal(fmt.Sprintf("Nonzero exit: %d", n), cause, line),
		n,
	}
}

var UnitVal = &UnitObject{}

func ExnVal(s string, cause string, line int) *ExnObject {
	entry := StackEntry{cause, line}
	return &ExnObject{val: s, stack: []StackEntry{entry}}
}

var NoExnVal = &ExnObject{}

func GetString(o Object) (string, error) {
	s, ok := o.(*StringObject)
	if !ok {
		return "", fmt.Errorf("Trying to use value of type '%s' as string", o.Type())
	}
	return s.val, nil
}

func GetBool(o Object) bool {
	n, ok := o.(*IntObject)
	if !ok {
		panic(fmt.Sprintf("Trying to use value of type '%s' as bool", o.Type()))
	}
	return n.val != 0
}
