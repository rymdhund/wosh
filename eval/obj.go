package eval

import "fmt"

// This will probably be an interface
type Object struct {
	Type  string
	Value interface{}
}

func (o Object) String() string {
	return fmt.Sprintf("%s(%v)", o.Type, o.Value)
}

func (o Object) add(o2 Object) Object {
	if o.Type != "int" {
		panic("trying to add non-integer")
	}
	if o2.Type != "int" {
		panic("trying to add non-integer")
	}
	return Object{"int", o.Value.(int) + o2.Value.(int)}
}

func IntVal(n int) Object {
	return Object{Type: "int", Value: n}
}

func StrVal(s string) Object {
	return Object{Type: "str", Value: s}
}

var UnitVal = Object{Type: "()", Value: nil}
