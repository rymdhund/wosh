package builtin

import (
	"fmt"
	"strconv"
	"unicode/utf8"

	. "github.com/rymdhund/wosh/obj"
)

func Add(o1, o2 Object) Object {
	switch t1 := o1.(type) {
	case *IntObject:
		i2, ok := o2.(*IntObject)
		if !ok {
			panic(fmt.Sprintf("trying to add %s and %s", t1.Class().Name, o2.Class().Name))
		}
		return IntVal(t1.Val + i2.Val)
	case *StringObject:
		t2, ok := o2.(*StringObject)
		if !ok {
			panic(fmt.Sprintf("trying to add %s and %s", t1.Class().Name, o2.Class().Name))
		}
		return StrVal(t1.Val + t2.Val)
	case *ListObject:
		t2, ok := o2.(*ListObject)
		if !ok {
			panic(fmt.Sprintf("trying to add %s and %s", t1.Class().Name, o2.Class().Name))
		}
		return t1.Concat(t2)
	default:
		panic(fmt.Sprintf("trying to add %s and %s", t1.Class().Name, o2.Class().Name))
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
		return StrVal(o.String())
	}
	return StrVal(strconv.Itoa(i.Val))
}

func Int(o Object) Object {
	s, ok := o.(*StringObject)
	if !ok {
		panic("trying to int non-string")
	}
	i, err := strconv.Atoi(s.Val)
	if err != nil {
		return UnitVal
	}
	return IntVal(i)
}

func Ord(o Object) Object {
	s, ok := o.(*StringObject)
	if !ok {
		panic("trying to ord non-string")
	}
	codes := []rune(s.Val)
	if len(codes) != 1 {
		panic("trying to ord string of len != 1")
	}
	return IntVal(int(codes[0]))
}

func Get(o Object, idx Object) (Object, bool) {
	switch t1 := o.(type) {
	case *ListObject:
		i, ok := idx.(*IntObject)
		if !ok {
			panic("Trying to get() non-integer index")
		}
		return t1.Get(i.Val)
	case *StringObject:
		i, ok := idx.(*IntObject)
		if !ok {
			panic("Trying to get() non-integer index")
		}
		runes := []rune(t1.Val)
		if i.Val >= len(runes) || i.Val < 0 {
			return UnitVal, false
		}
		c := string(runes[i.Val])
		return StrVal(c), true
	case *MapObject:
		s, ok := idx.(*StringObject)
		if !ok {
			panic("Trying to map.get() non-string key")
		}
		return t1.Get(s.Val)
	default:
		panic(fmt.Sprintf("Trying to get() on %s object", o.Class().Name))
	}
}

// idx1 and idx2 can be nil for empty indexes
func Slice(o Object, idx1, idx2, step Object) Object {
	i1, ok := idx1.(*IntObject)
	if !ok {
		if idx1 == nil {
			i1 = IntVal(0)
		} else {
			panic("Trying to slice() non-integer index")
		}
	}
	i2, ok := idx2.(*IntObject)
	if !ok {
		if idx2 == nil {
			i2 = Len(o)
		} else {
			panic("Trying to slice() non-integer index")
		}
	}
	istep, ok := step.(*IntObject)
	if !ok {
		panic("Trying to slice() non-integer index")
	}

	switch t1 := o.(type) {
	case *ListObject:
		return t1.Slice(i1, i2, istep)
	case *StringObject:
		return t1.Slice(i1, i2, istep)
	default:
		panic("Trying to slice() on non-compatible object")
	}

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
			panic(fmt.Sprintf("trying to compare %s and %s", t1.Class().Name, o2.Class().Name))
		}
		return BoolVal(t1.Val <= i2.Val)
	case *StringObject:
		s2, ok := o2.(*StringObject)
		if !ok {
			panic(fmt.Sprintf("trying to compare %s and %s", t1.Class().Name, o2.Class().Name))
		}
		return BoolVal(t1.Val <= s2.Val)
	default:
		panic(fmt.Sprintf("Trying to compare %s and %s", t1.Class().Name, o2.Class().Name))
	}
}

func Less(o1, o2 Object) Object {
	switch t1 := o1.(type) {
	case *IntObject:
		i2, ok := o2.(*IntObject)
		if !ok {
			panic(fmt.Sprintf("trying to compare %s and %s", t1.Class().Name, o2.Class().Name))
		}
		return BoolVal(t1.Val < i2.Val)
	case *StringObject:
		s2, ok := o2.(*StringObject)
		if !ok {
			panic(fmt.Sprintf("trying to compare %s and %s", t1.Class().Name, o2.Class().Name))
		}
		return BoolVal(t1.Val < s2.Val)
	default:
		panic(fmt.Sprintf("Trying to compare %s and %s", t1.Class().Name, o2.Class().Name))
	}
}

func Greater(o1, o2 Object) Object {
	switch t1 := o1.(type) {
	case *IntObject:
		i2, ok := o2.(*IntObject)
		if !ok {
			panic(fmt.Sprintf("trying to compare %s and %s", t1.Class().Name, o2.Class().Name))
		}
		return BoolVal(t1.Val > i2.Val)
	case *StringObject:
		s2, ok := o2.(*StringObject)
		if !ok {
			panic(fmt.Sprintf("trying to compare %s and %s", t1.Class().Name, o2.Class().Name))
		}
		return BoolVal(t1.Val > s2.Val)
	default:
		panic(fmt.Sprintf("Trying to compare %s and %s", t1.Class().Name, o2.Class().Name))
	}
}

func GreaterEq(o1, o2 Object) Object {
	switch t1 := o1.(type) {
	case *IntObject:
		i2, ok := o2.(*IntObject)
		if !ok {
			panic(fmt.Sprintf("trying to compare %s and %s", t1.Class().Name, o2.Class().Name))
		}
		return BoolVal(t1.Val >= i2.Val)
	case *StringObject:
		s2, ok := o2.(*StringObject)
		if !ok {
			panic(fmt.Sprintf("trying to compare %s and %s", t1.Class().Name, o2.Class().Name))
		}
		return BoolVal(t1.Val >= s2.Val)
	default:
		panic(fmt.Sprintf("Trying to compare %s and %s", t1.Class().Name, o2.Class().Name))
	}
}

func Len(o Object) *IntObject {
	switch t := o.(type) {
	case *StringObject:
		return IntVal(utf8.RuneCountInString(t.Val))
	case *ListObject:
		return IntVal(t.Len())
	case *MapObject:
		return IntVal(len(t.Map))
	default:
		panic(fmt.Sprintf("Trying to get length of %s", o.Class().Name))
	}
}

func BoolAnd(o1, o2 Object) Object {
	b1, ok := o1.(*BoolObject)
	if !ok {
		panic("trying to && non-bool")
	}
	b2, ok := o2.(*BoolObject)
	if !ok {
		panic("trying to && non-bool")
	}
	return BoolVal(b1.Val && b2.Val)
}

func BoolOr(o1, o2 Object) Object {
	b1, ok := o1.(*BoolObject)
	if !ok {
		panic("trying to || non-bool")
	}
	b2, ok := o2.(*BoolObject)
	if !ok {
		panic("trying to || non-bool")
	}
	return BoolVal(b1.Val || b2.Val)
}

func BoolNot(o Object) Object {
	i, ok := o.(*BoolObject)
	if !ok {
		panic("trying to not non-bool")
	}
	return BoolVal(!i.Val)
}

func Cons(o1, o2 Object) Object {
	switch t2 := o2.(type) {
	case *ListObject:
		return ListVal(o1, t2)
	default:
		panic(fmt.Sprintf("trying to cons %s and %s", o1.Class().Name, o2.Class().Name))
	}
}

func Set(o, key, val Object) {
	switch t := o.(type) {
	case *MapObject:
		s, ok := key.(*StringObject)
		if !ok {
			panic("trying to map.set() non-string key")
		}
		t.Map[s.Val] = val
	default:
		panic(fmt.Sprintf("trying to Set(%s, %s, %s)", o.Class().Name, key.Class().Name, val.Class().Name))
	}
}
