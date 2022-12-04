package interpret

import "fmt"

// Return value if we succeed, return nil if we don't have add for this value
func builtinAdd(a, b Value) (Value, error) {
	switch l := a.(type) {
	case *IntValue:
		r, ok := b.(*IntValue)
		if !ok {
			return nil, fmt.Errorf("Trying to add %s and %s", a.Type().Name, b.Type().Name)
		} else {
			return NewInt(l.Val + r.Val), nil
		}
	case *StringValue:
		r, ok := b.(*StringValue)
		if !ok {
			return nil, fmt.Errorf("Trying to add %s and %s", a.Type().Name, b.Type().Name)
		} else {
			return NewString(l.Val + r.Val), nil
		}
	case *ListValue:
		r, ok := b.(*ListValue)
		if !ok {
			return nil, fmt.Errorf("Trying to add %s and %s", a.Type().Name, b.Type().Name)
		} else {
			return l.Concat(r), nil
		}
	default:
		return nil, nil
	}
}

// Returns nil if we don't have eq implementation for the type
func builtinEq(a, b Value) *BoolValue {
	switch t := a.(type) {
	case *IntValue:
		i2, ok := b.(*IntValue)
		if !ok {
			return NewBool(false)
		} else {
			return NewBool(t.Val == i2.Val)
		}
	case *StringValue:
		s2, ok := b.(*StringValue)
		if !ok {
			return NewBool(false)
		} else {
			return NewBool(t.Val == s2.Val)
		}
	case *BoolValue:
		b2, ok := b.(*BoolValue)
		if !ok {
			return NewBool(false)
		} else {
			return NewBool(t.Val == b2.Val)
		}
	case *NilValue:
		_, ok := b.(*NilValue)
		return NewBool(ok)
	default:
		return nil
	}
}
