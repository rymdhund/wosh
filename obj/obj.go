package obj

import (
	"fmt"
	"reflect"

	"github.com/rymdhund/wosh/ast"
)

// An entry in a stack trace
type StackEntry struct {
	Function string
	Line     int
}

type Object interface {
	Type() string
	Eq(Object) bool
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

func (t *StringObject) Eq(o Object) bool {
	x, ok := o.(*StringObject)
	if !ok {
		return false
	}
	return x.Val == t.Val
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

func (t *IntObject) Eq(o Object) bool {
	x, ok := o.(*IntObject)
	if !ok {
		return false
	}
	return x.Val == t.Val
}

type BoolObject struct {
	Val bool
}

func (t *BoolObject) Type() string {
	return "bool"
}

func (t *BoolObject) String() string {
	return fmt.Sprintf("%s(%v)", t.Type(), t.Val)
}

func (t *BoolObject) Eq(o Object) bool {
	x, ok := o.(*BoolObject)
	if !ok {
		return false
	}
	return x.Val == t.Val
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

func (t *ExnObject) Eq(o Object) bool {
	x, ok := o.(*ExnObject)
	if !ok {
		return false
	}
	return x == t
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

func (t *UnitObject) Eq(o Object) bool {
	_, ok := o.(*UnitObject)
	return ok
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

func (t *ListObject) Eq(o Object) bool {
	x, ok := o.(*ListObject)
	if !ok {
		return false
	}
	a := t.head
	b := x.head

	for true {
		if a == nil {
			return b == nil
		} else if b == nil {
			return false
		} else if !a.Val.Eq(b.Val) {
			return false
		}
		a = a.next
		b = b.next
	}

	// unrearchable
	return false
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

func (t *ListObject) Len() int {
	cnt := 0

	cur := t.head
	for cur != nil {
		cnt += 1
		cur = cur.next
	}

	return cnt
}

type FunctionObject struct {
	Expr *ast.FuncExpr
}

func (f *FunctionObject) Type() string {
	return "func"
}

func (t *FunctionObject) Eq(o Object) bool {
	x, ok := o.(*FunctionObject)
	if !ok {
		return false
	}
	return x == t
}

func IntVal(n int) *IntObject {
	return &IntObject{Val: n}
}

func BoolVal(b bool) *BoolObject {
	return &BoolObject{Val: b}
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
	n, ok := o.(*BoolObject)
	if !ok {
		panic(fmt.Sprintf("Trying to use value of type '%s' as bool", o.Type()))
	}
	return n.Val
}
