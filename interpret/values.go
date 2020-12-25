package interpret

import (
	"fmt"
	"os"
)

type QuickMethod uint8

const (
	METH_EQ = iota
)

type FunctionMap map[string]*FunctionValue

type Type struct {
	Name    string
	Methods FunctionMap
}

var NilType = &Type{"Nil", FunctionMap{}}
var BoolType = &Type{"Bool", FunctionMap{}}
var IntType = &Type{"Int", FunctionMap{}}
var StringType = &Type{"Str", FunctionMap{}}
var ListType = &Type{"List", FunctionMap{}}
var MapType = &Type{"Map", FunctionMap{}}
var FunctionType = &Type{"Function", FunctionMap{}}
var ClosureType = &Type{"Closure", FunctionMap{}}
var ExceptionType = &Type{"Exception", FunctionMap{}}
var BoxType = &Type{"Box", FunctionMap{}}

type Value interface {
	Type() *Type
	String() string
}

func Equal(v1, v2 Value) bool {
	switch x1 := v1.(type) {
	case *IntValue:
		x2, ok := v2.(*IntValue)
		if !ok {
			return false
		}
		return x1.Val == x2.Val
	case *StringValue:
		x2, ok := v2.(*StringValue)
		if !ok {
			return false
		}
		return x1.Val == x2.Val
	case *NilValue:
		_, ok := v2.(*NilValue)
		if !ok {
			return false
		}
		return true
	}
	return false
}

type FunctionValue struct {
	Name  string
	Arity int
	Chunk *Chunk

	// Slot indexes that need to be captured when creating a closure from this function
	OuterCaptures []uint8

	// The slots to put captured variables in when calling this function
	CaptureSlots []uint8

	SlotsToPutOnHeap []uint8
}

func (t *FunctionValue) Type() *Type {
	return FunctionType
}

func (t *FunctionValue) String() string {
	return fmt.Sprintf("%s(%s, %d)", t.Type().Name, t.Name, t.Arity)
}

func (t *FunctionValue) DebugPrint() {
	t.Chunk.disassemble(t.Name, os.Stdout)
}

type ClosureValue struct {
	Function *FunctionValue
	Captures []*BoxValue
}

func (t *ClosureValue) Type() *Type {
	return ClosureType
}

func (t *ClosureValue) String() string {
	return t.Function.String()
}

func (t *ClosureValue) DebugPrint() {
	t.Function.DebugPrint()
}

func NewClosure(fn *FunctionValue, captures []*BoxValue) *ClosureValue {
	return &ClosureValue{fn, captures}
}

type StringValue struct {
	Val string
}

func (t *StringValue) Type() *Type {
	return StringType
}

func (t *StringValue) String() string {
	return fmt.Sprintf("%s(%v)", t.Type().Name, t.Val)
}

func (t *StringValue) Len() int {
	return len([]rune(t.Val))
}

func (t *StringValue) Slice(i, j, step *IntValue) *StringValue {
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
		return NewString("")
	}
	if idx1 < 0 {
		idx1 = 0
	}
	if idx2 > length {
		idx2 = length
	}

	return NewString(string([]rune(t.Val[idx1:idx2])))
}

type IntValue struct {
	Val int
}

func (t *IntValue) Type() *Type {
	return IntType
}

func (t *IntValue) String() string {
	return fmt.Sprintf("%s(%v)", t.Type().Name, t.Val)
}

type BoolValue struct {
	Val bool
}

func (t *BoolValue) Type() *Type {
	return BoolType
}

func (t *BoolValue) String() string {
	return fmt.Sprintf("%s(%v)", t.Type().Name, t.Val)
}

type StackEntry struct {
	Function string
	Line     int
}

type ExnValue struct {
	Val   string
	stack []StackEntry
}

func (t *ExnValue) Type() *Type {
	return ExceptionType
}

func (t *ExnValue) String() string {
	return fmt.Sprintf("%s(%v)", t.Type().Name, t.Val)
}

func (t *ExnValue) Msg() string {
	return t.Val
}

func (t *ExnValue) AddStackEntry(entry StackEntry) {
	t.stack = append(t.stack, entry)
}

func (t *ExnValue) GetStackTrace() string {
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

type NilValue struct {
}

func (t *NilValue) Type() *Type {
	return NilType
}

func (t *NilValue) String() string {
	return "nil"
}

type ListNode struct {
	Val  Value
	next *ListNode
}

type ListValue struct {
	head *ListNode
	len  int
}

func (t *ListValue) Type() *Type {
	return ListType
}

func (t *ListValue) String() string {
	return "list"
}

// Returns (nil, false) in case of out of bounds error
func (t *ListValue) Get(idx int) (Value, bool) {
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
func (t *ListValue) PrivPush(o Value) {
	t.len++
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

func (t *ListValue) Concat(o *ListValue) *ListValue {
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
	return &ListValue{head: copyHead, len: t.len + o.len}
}

func (t *ListValue) Len() int {
	return t.len
}

func (t *ListValue) Slice(i, j, step *IntValue) *ListValue {
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

func NewInt(n int) *IntValue {
	return &IntValue{Val: n}
}

func NewBool(b bool) *BoolValue {
	return &BoolValue{Val: b}
}

func NewString(s string) *StringValue {
	return &StringValue{Val: s}
}

var Nil = &NilValue{}

func NewExn(s string, cause string, line int) *ExnValue {
	entry := StackEntry{cause, line}
	return &ExnValue{Val: s, stack: []StackEntry{entry}}
}

func ListCons(val Value, tail *ListValue) *ListValue {
	node := ListNode{Val: val, next: tail.head}
	return &ListValue{head: &node, len: tail.len + 1}
}

func ListNil() *ListValue {
	return &ListValue{head: nil, len: 0}
}

var NoExnVal = &ExnValue{}

func GetString(v Value) (string, error) {
	s, ok := v.(*StringValue)
	if !ok {
		return "", fmt.Errorf("Trying to use value of type '%s' as string", v.Type().Name)
	}
	return s.Val, nil
}

func GetBool(v Value) bool {
	n, ok := v.(*BoolValue)
	if !ok {
		panic(fmt.Sprintf("Trying to use value of type '%s' as bool", v.Type().Name))
	}
	return n.Val
}

type MapValue struct {
	Map map[string]Value
}

func (t *MapValue) Type() *Type {
	return MapType
}

// Returns (nil, false) in case of out of bounds error
func (t *MapValue) Get(key string) (Value, bool) {
	res, ok := t.Map[key]
	if !ok {
		return Nil, true
	}
	return res, ok
}

func NewMap() *MapValue {
	return &MapValue{map[string]Value{}}
}

type BoxValue struct {
	Val Value
}

func (t *BoxValue) Type() *Type {
	return BoxType
}

func (t *BoxValue) String() string {
	return fmt.Sprintf("Box[%s]", t.Val.String())
}

func (t *BoxValue) Get() Value {
	return t.Val
}

func (t *BoxValue) Set(v Value) {
	t.Val = v
}
