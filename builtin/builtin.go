package builtin

import (
	"fmt"
	"strconv"

	. "github.com/rymdhund/wosh/obj"
)

func Add(o1, o2 Object) Object {
	switch t1 := o1.(type) {
	case *IntObject:
		i2, ok := o2.(*IntObject)
		if !ok {
			panic(fmt.Sprintf("trying to add %s and %s", t1.Type(), o2.Type()))
		}
		return IntVal(t1.Val + i2.Val)
	case *StringObject:
		i2, ok := o2.(*StringObject)
		if !ok {
			panic(fmt.Sprintf("trying to add %s and %s", t1.Type(), o2.Type()))
		}
		return StrVal(t1.Val + i2.Val)
	default:
		panic(fmt.Sprintf("trying to add %s and %s", t1.Type(), o2.Type()))
	}
}

func Sub(o1, o2 Object) Object {
	i1, ok := o1.(*IntObject)
	if !ok {
		panic("trying to sub non-integer")
	}
	i2, ok := o2.(*IntObject)
	if !ok {
		panic("trying to sub non-integer")
	}
	return IntVal(i1.Val - i2.Val)
}

func Mult(o1, o2 Object) Object {
	i1, ok := o1.(*IntObject)
	if !ok {
		panic("trying to mult non-integer")
	}
	i2, ok := o2.(*IntObject)
	if !ok {
		panic("trying to mult non-integer")
	}
	return IntVal(i1.Val * i2.Val)
}

func Div(o1, o2 Object) Object {
	i1, ok := o1.(*IntObject)
	if !ok {
		panic("trying to div non-integer")
	}
	i2, ok := o2.(*IntObject)
	if !ok {
		panic("trying to div non-integer")
	}
	return IntVal(i1.Val / i2.Val)
}

func Neg(o Object) Object {
	i, ok := o.(*IntObject)
	if !ok {
		panic("trying to negate non-integer")
	}
	return IntVal(-i.Val)
}

func Str(o Object) *StringObject {
	i, ok := o.(*IntObject)
	if !ok {
		panic("trying to str non-integer")
	}
	return StrVal(strconv.Itoa(i.Val))
}

func Get(o Object, idx Object) (Object, bool) {
	lst, ok := o.(*ListObject)
	if !ok {
		panic("trying to get() on non-list")
	}

	i, ok := idx.(*IntObject)
	if !ok {
		panic("trying to get() non-integer index")
	}
	return lst.Get(i.Val)
}
