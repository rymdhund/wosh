package obj

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/rymdhund/wosh/ast"
)

// An entry in a stack trace
type StackEntry struct {
	Function string
	Line     int
}

type Object interface {
	String() string
	Eq(Object) bool
	Class() Class
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

func (t *StringObject) Class() Class {
	return StringClass
}

func (t *StringObject) String() string {
	return fmt.Sprintf("%s(%v)", t.Class().Name, t.Val)
}

func (t *StringObject) Eq(o Object) bool {
	x, ok := o.(*StringObject)
	if !ok {
		return false
	}
	return x.Val == t.Val
}

func (t *StringObject) Len() int {
	return len([]rune(t.Val))
}

func (t *StringObject) Slice(i, j, step *IntObject) *StringObject {
	if step.Val == 0 {
		panic("Cannot slice on step = 0")
	}
	if step.Val != 1 {
		// TODO
		panic("Not yet support for different step sizes")
	}

	length := t.Len()
	idx1 := i.Val
	idx2 := j.Val
	if idx1 < 0 {
		idx1 = length + idx1
	}
	if idx2 < 0 {
		idx2 = t.Len() + idx2
	}
	if idx2 <= idx1 || idx1 >= length || idx2 <= 0 {
		return StrVal("")
	}
	if idx1 < 0 {
		idx1 = 0
	}
	if idx2 > length {
		idx2 = length
	}

	return StrVal(string([]rune(t.Val[idx1:idx2])))
}

type IntObject struct {
	Val int
}

func (t *IntObject) Class() Class {
	return IntClass
}

func (t *IntObject) String() string {
	return fmt.Sprintf("%s(%v)", t.Class().Name, t.Val)
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

func (t *BoolObject) Class() Class {
	return BoolClass
}

func (t *BoolObject) String() string {
	return fmt.Sprintf("%s(%v)", t.Class().Name, t.Val)
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

func (t *ExnObject) Class() Class {
	return ExceptionClass
}

func (t *ExnObject) String() string {
	return fmt.Sprintf("%s(%v)", t.Class().Name, t.Val)
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

type UnitObject struct {
}

func (t *UnitObject) Class() Class {
	return UnitClass
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

func (t *ListObject) Class() Class {
	return ListClass
}

func (t *ListObject) String() string {
	values := []string{}
	x := t.head
	for x != nil {
		values = append(values, x.Val.String())
		x = x.next
	}
	return "[" + strings.Join(values, ", ") + "]"
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

// mutates the list! only for internal usage
func (t *ListObject) PrivPush(o Object) {
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

func (t *ListObject) Concat(o *ListObject) *ListObject {
	if t.head == nil {
		return o
	}

	copyHead := &ListNode{Val: t.head.Val, next: nil}
	copyCur := copyHead

	cur := t.head
	for cur.next != nil {
		cur = cur.next
		copyCur.next = &ListNode{Val: cur.Val, next: nil}
		copyCur = copyCur.next
	}
	copyCur.next = o.head
	return &ListObject{head: copyHead}
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

func (t *ListObject) Slice(i, j, step *IntObject) *ListObject {
	if step.Val == 0 {
		panic("List cannot slice on step = 0")
	}
	if step.Val != 1 {
		// TODO
		panic("Not yet support for different step sizes")
	}

	length := t.Len()
	idx1 := i.Val
	idx2 := j.Val
	if idx1 < 0 {
		idx1 = length + idx1
	}
	if idx2 < 0 {
		idx2 = t.Len() + idx2
	}
	if idx2 <= idx1 || idx1 >= length || idx2 <= 0 {
		return ListNil()
	}

	cnt := 0
	cur := t.head
	for cnt < idx1 && cur != nil {
		cur = cur.next
		cnt++
	}
	list := ListNil()
	for cnt < idx2 && cur != nil {
		list.PrivPush(cur.Val)
		cur = cur.next
		cnt++
	}
	return list
}

type FunctionObject struct {
	Expr *ast.FuncDefExpr
}

func (f *FunctionObject) Class() Class {
	return FunctionClass
}

func (f *FunctionObject) String() string {
	return "<func>"
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
		return "", fmt.Errorf("Trying to use value of type '%s' as string", o.Class().Name)
	}
	return s.Val, nil
}

func GetBool(o Object) bool {
	n, ok := o.(*BoolObject)
	if !ok {
		panic(fmt.Sprintf("Trying to use value of type '%s' as bool", o.Class().Name))
	}
	return n.Val
}

type MapObject struct {
	Map map[string]Object
}

func (t *MapObject) Class() Class {
	return MapClass
}

func (t *MapObject) String() string {
	values := []string{}
	for key, val := range t.Map {
		values = append(values, fmt.Sprintf("%s: %s", key, val.String()))
	}
	return "{" + strings.Join(values, ", ") + "}"
}

func (t *MapObject) Eq(o Object) bool {
	// TODO
	return false
}

// Returns (nil, false) in case of out of bounds error
func (t *MapObject) Get(key string) (Object, bool) {
	res, ok := t.Map[key]
	return res, ok
}

func NewMap() *MapObject {
	return &MapObject{map[string]Object{}}
}
