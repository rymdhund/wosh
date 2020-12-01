package obj

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
	Val string
}

func (t *StringObject) Type() string {
	return "str"
}

func (t *StringObject) String() string {
	return fmt.Sprintf("%s(%v)", t.Type(), t.Val)
}

type IntObject struct {
	Val int
}

func (t *IntObject) Type() string {
	return "int"
}

func (t *IntObject) String() string {
	return fmt.Sprintf("%s(%v)", t.Type(), t.Val)
}

type ExnObject struct {
	Val   string
	stack []StackEntry
}

func (t *ExnObject) Type() string {
	return "exception"
}

func (t *ExnObject) String() string {
	return fmt.Sprintf("%s(%v)", t.Type(), t.Val)
}

func (t *ExnObject) Msg() string {
	return t.Val
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

type ListNode struct {
	Val  Object
	next *ListNode
}

type ListObject struct {
	head *ListNode
}

func (t *ListObject) Type() string {
	return "list"
}

func (t *ListObject) String() string {
	return "[]"
}

// Returns (nil, false) in case of out of bounds error
func (t *ListObject) Get(idx int) (Object, bool) {
	if idx < 0 {
		return nil, false
	}

	cur := t.head
	for true {
		if cur == nil {
			return nil, false
		}

		if idx == 0 {
			return cur.Val, true
		}

		cur = cur.next
		idx--
	}

	// unreachable
	return nil, false
}

func (t *ListObject) Add(o Object) {
	e := &ListNode{Val: o, next: nil}
	if t.head == nil {
		t.head = e
		return
	}

	cur := t.head
	for cur.next != nil {
		cur = cur.next
	}
	cur.next = e
}

func IntVal(n int) *IntObject {
	return &IntObject{Val: n}
}

func StrVal(s string) *StringObject {
	return &StringObject{Val: s}
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
	return &ExnObject{Val: s, stack: []StackEntry{entry}}
}

func ListVal(val Object, tail *ListObject) *ListObject {
	node := ListNode{Val: val, next: tail.head}
	return &ListObject{head: &node}
}

func ListNil() *ListObject {
	return &ListObject{head: nil}
}

var NoExnVal = &ExnObject{}

func GetString(o Object) (string, error) {
	s, ok := o.(*StringObject)
	if !ok {
		return "", fmt.Errorf("Trying to use value of type '%s' as string", o.Type())
	}
	return s.Val, nil
}

func GetBool(o Object) bool {
	n, ok := o.(*IntObject)
	if !ok {
		panic(fmt.Sprintf("Trying to use value of type '%s' as bool", o.Type()))
	}
	return n.Val != 0
}
