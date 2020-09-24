package eval

import (
	"fmt"
	"reflect"
)

type Object interface {
	Type() string
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

type ExitObject struct {
	val int
}

func (t *ExitObject) Type() string {
	return "exit"
}

func (t *ExitObject) String() string {
	return fmt.Sprintf("%s(%v)", t.Type(), t.val)
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

func IntVal(n int) *IntObject {
	return &IntObject{val: n}
}

func StrVal(s string) *StringObject {
	return &StringObject{val: s}
}

func ExitVal(n int) *ExitObject {
	return &ExitObject{val: n}
}

var UnitVal = &UnitObject{}

func GetString(o Object) string {
	s, ok := o.(*StringObject)
	if !ok {
		panic(fmt.Sprintf("Trying to use value of type '%s' as string", o.Type()))
	}
	return s.val
}

func GetBool(o Object) bool {
	n, ok := o.(*IntObject)
	if !ok {
		panic(fmt.Sprintf("Trying to use value of type '%s' as bool", o.Type()))
	}
	return n.val != 0
}
