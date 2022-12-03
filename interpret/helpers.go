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
