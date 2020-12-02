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
	i, ok := idx.(*IntObject)
	if !ok {
		panic("Trying to get() non-integer index")
	}

	lst, ok := o.(*ListObject)
	if ok {
		return lst.Get(i.Val)
	}
	str, ok := o.(*StringObject)
	if ok {
		if i.Val >= len(str.Val) || i.Val < 0 {
			return UnitVal, false
		}
		c := string(str.Val[i.Val])
		return StrVal(c), true
	}
	panic("Trying to get() on non-compatible object")

}

func Eq(o1, o2 Object) Object {
	return BoolVal(o1.Eq(o2))
}

func Neq(o1, o2 Object) Object {
	return BoolVal(!o1.Eq(o2))
}

func LessEq(o1, o2 Object) Object {
	switch t1 := o1.(type) {
	case *IntObject:
		i2, ok := o2.(*IntObject)
		if !ok {
			panic(fmt.Sprintf("trying to compare %s and %s", t1.Type(), o2.Type()))
		}
		return BoolVal(t1.Val <= i2.Val)
	default:
		panic(fmt.Sprintf("Trying to compare %s and %s", t1.Type(), o2.Type()))
	}
}

func Less(o1, o2 Object) Object {
	switch t1 := o1.(type) {
	case *IntObject:
		i2, ok := o2.(*IntObject)
		if !ok {
			panic(fmt.Sprintf("trying to compare %s and %s", t1.Type(), o2.Type()))
		}
		return BoolVal(t1.Val < i2.Val)
	default:
		panic(fmt.Sprintf("Trying to compare %s and %s", t1.Type(), o2.Type()))
	}
}

func Greater(o1, o2 Object) Object {
	switch t1 := o1.(type) {
	case *IntObject:
		i2, ok := o2.(*IntObject)
		if !ok {
			panic(fmt.Sprintf("trying to compare %s and %s", t1.Type(), o2.Type()))
		}
		return BoolVal(t1.Val > i2.Val)
	default:
		panic(fmt.Sprintf("Trying to compare %s and %s", t1.Type(), o2.Type()))
	}
}

func GreaterEq(o1, o2 Object) Object {
	switch t1 := o1.(type) {
	case *IntObject:
		i2, ok := o2.(*IntObject)
		if !ok {
			panic(fmt.Sprintf("trying to compare %s and %s", t1.Type(), o2.Type()))
		}
		return BoolVal(t1.Val >= i2.Val)
	default:
		panic(fmt.Sprintf("Trying to compare %s and %s", t1.Type(), o2.Type()))
	}
}
