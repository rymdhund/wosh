package interpret

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"unicode/utf8"
)

const FRAMES_MAX = 64
const STACK_MAX = 256
const DEBUG_TRACE = false

type VM struct {
	//frames       [FRAMES_MAX]CallFrame
	frameCount   int // for debug purposes
	currentFrame *CallFrame
	globals      map[string]Value
}

type CallFrame struct {
	closure     *ClosureValue
	ip          int
	code        []Op // points to chunk code
	stack       [STACK_MAX]Value
	stackTop    int        // points to next unused element in stack
	returnFrame *CallFrame // which frame to return to
	returnIp    int
	//resumeFrame int // which frame to resume to

	// Handlers for effects
	handlers []Handler
}

type Handler struct {
	effect string
	//handler  *ClosureValue
	//doneLine int // line to land at after handler returns
	frame *CallFrame
	ip    int // instruction pointer
}

func builtinReadlines(file Value) Value {
	filename := file.(*StringValue).Val
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(fmt.Sprintf("Can't read file: %s", filename))
	}
	lines := strings.Split(strings.Trim(string(content), "\n"), "\n")

	list := ListNil()
	for i := len(lines) - 1; i >= 0; i-- {
		list = ListCons(NewString(lines[i]), list)
	}
	return list
}

func builtinLen(v Value) Value {
	switch x := v.(type) {
	case *ListValue:
		return NewInt(x.len)
	case *MapValue:
		return NewInt(len(x.Map))
	case *StringValue:
		return NewInt(utf8.RuneCountInString(x.Val))
	default:
		panic(fmt.Sprintf("%v does not support len()", v))
	}
}

func builtinStr(value Value) Value {
	return NewString(value.String())
}

func builtinPrintln(value Value) Value {
	switch v := value.(type) {
	case *StringValue:
		println(v.Val)
	default:
		println(value.String())
	}
	return Nil
}

func builtinAtoi(value Value) Value {
	i, err := strconv.Atoi(value.(*StringValue).Val)
	if err != nil {
		panic(err)
	}
	return NewInt(i)
}

func builtinOrd(value Value) Value {
	s := []rune(value.(*StringValue).Val)
	if len(s) != 1 {
		panic("Ord expected string of length 1")
	}
	return NewInt(int(s[0]))
}

func builtinAssert(value Value, message Value) Value {
	if !value.(*BoolValue).Val {
		panic(fmt.Sprintf("Assertion error: %s", message))
	}
	return Nil
}

func builtinItems(v Value) Value {
	switch x := v.(type) {
	case *MapValue:
		pairs := ListNil()
		for k, v := range x.Map {
			pairs = ListCons(ListCons(NewString(k), ListCons(v, ListNil())), pairs)
		}
		return pairs
	default:
		panic(fmt.Sprintf("%v does not support items()", v))
	}
}

func builtinTypeof(v Value) Value {
	return NewTypeValue(v.Type())
}

func NewVm() *VM {
	globals := map[string]Value{}
	globals["readlines"] = NewBuiltin("readlines", 1, builtinReadlines)
	globals["str"] = NewBuiltin("str", 1, builtinStr)
	globals["println"] = NewBuiltin("println", 1, builtinPrintln)
	globals["atoi"] = NewBuiltin("atoi", 1, builtinAtoi)
	globals["len"] = NewBuiltin("len", 1, builtinLen)
	globals["ord"] = NewBuiltin("ord", 1, builtinOrd)
	globals["assert"] = NewBuiltin("assert", 2, builtinAssert)
	globals["items"] = NewBuiltin("items", 1, builtinItems)
	globals["typeof"] = NewBuiltin("typeof", 1, builtinTypeof)

	globals["Nil"] = NewTypeValue(BoolType)
	globals["Bool"] = NewTypeValue(BoolType)
	globals["Int"] = NewTypeValue(IntType)
	globals["Str"] = NewTypeValue(StringType)
	globals["List"] = NewTypeValue(ListType)
	globals["Map"] = NewTypeValue(MapType)

	return &VM{globals: globals}
}

func (frame *CallFrame) pushStack(v Value) {
	frame.stack[frame.stackTop] = v
	frame.stackTop += 1
}

func (frame *CallFrame) popStack() Value {
	frame.stackTop -= 1
	return frame.stack[frame.stackTop]
}

func (frame *CallFrame) peekStack(offset int) Value {
	return frame.stack[frame.stackTop-1-offset]
}

func (frame *CallFrame) replaceStack(offset int, v Value) {
	frame.stack[frame.stackTop-1-offset] = v
}

func (vm *VM) NewFrame(cl *ClosureValue, args []Value, returnFrame *CallFrame, returnIp int) *CallFrame {
	frame := &CallFrame{} //&vm.frames[vm.currentFrame]
	frame.closure = cl
	frame.ip = 0
	frame.pushStack(cl)
	for _, arg := range args {
		frame.pushStack(arg)
	}
	for i := 0; i < len(cl.Function.Chunk.LocalNames)-len(args)-1; i++ {
		frame.pushStack(Nil)
	}
	for _, x := range cl.Function.SlotsToPutOnHeap {
		_, ok := frame.stack[x].(*BoxValue)
		if !ok {
			frame.stack[x] = &BoxValue{frame.stack[x]}
		}
	}
	for i, x := range cl.Function.CaptureSlots {
		frame.stack[x] = cl.Captures[i]
	}
	frame.returnFrame = returnFrame
	frame.returnIp = returnIp
	vm.frameCount++
	return frame
}

func (vm *VM) Interpret(main *FunctionValue) (Value, error) {
	vm.frameCount = 0
	frame := vm.NewFrame(NewClosure(main, []*BoxValue{}), []Value{}, nil, -1)
	vm.currentFrame = frame
	return vm.run()
}

func (frame *CallFrame) readCode() Op {
	op := frame.closure.Function.Chunk.Code[frame.ip]
	frame.ip += 1
	return op
}

func (frame *CallFrame) readConstant() Value {
	pos := frame.readCode()
	return frame.closure.Function.Chunk.Constants[pos]
}

func (frame *CallFrame) readFunction() *FunctionValue {
	pos := frame.readCode()
	fn := frame.closure.Function.Chunk.Constants[pos].(*FunctionValue)
	return fn
}

func (frame *CallFrame) readName() string {
	pos := frame.readCode()
	return frame.closure.Function.Chunk.Names[pos]
}

func (frame *CallFrame) readUint16() uint16 {
	return uint16(frame.readCode())<<8 + uint16(frame.readCode())
}

func (frame *CallFrame) putSlot(idx uint8, v Value) {
	frame.stack[idx] = v
}

func (vm *VM) run() (Value, error) {
	for true {
		frame := vm.currentFrame

		startIP := frame.ip
		instr := frame.readCode()

		if DEBUG_TRACE {
			fmt.Printf("%-4d %-20s ", startIP, instr)
			for i := 0; i < frame.stackTop; i++ {
				fmt.Printf("[ %s ]", frame.stack[i].String())
			}
			fmt.Print("\n")
		}

		var err error
		switch instr {
		case OP_RETURN:
			retVal := frame.popStack()

			frame.stackTop -= len(frame.closure.Function.Chunk.LocalNames)
			// todo: clean up references for garbage collection

			if frame.stackTop != 0 {
				panic("expected empty stack")
			}

			// todo: clean up memory
			if frame.returnFrame == nil {
				return retVal, nil
			} else {
				frame.returnFrame.ip = frame.returnIp
				if DEBUG_TRACE {
					fmt.Printf("Returning to %d\n", frame.returnFrame.ip)
				}
				frame.returnFrame.pushStack(retVal)
				vm.currentFrame = frame.returnFrame
			}
		case OP_LOAD_CONSTANT:
			constant := frame.readConstant()
			frame.pushStack(constant)
		case OP_MAKE_CLOSURE:
			function := frame.readFunction()
			captures := []*BoxValue{}
			for _, i := range function.OuterCaptures {
				v := frame.stack[i].(*BoxValue)
				captures = append(captures, v)
			}
			closure := NewClosure(function, captures)
			frame.pushStack(closure)
		case OP_NOP:
			// do nothing
		case OP_NIL:
			frame.pushStack(Nil)
		case OP_TRUE:
			frame.pushStack(NewBool(true))
		case OP_FALSE:
			frame.pushStack(NewBool(false))
		case OP_EQ:
			ok := frame.opEq()
			if err == nil && !ok {
				err = vm.opCallMethod(1, "eq")
			}
		case OP_NEG:
			err = frame.opNeg()
		case OP_ADD:
			var ok bool
			ok, err = frame.opAdd()
			if err == nil && !ok {
				err = vm.opCallMethod(1, "add")
			}
		case OP_SUB:
			var ok bool
			ok, err = frame.opSub()
			if err == nil && !ok {
				err = vm.opCallMethod(1, "sub")
			}
		case OP_MULT:
			var ok bool
			ok, err = frame.opMult()
			if err == nil && !ok {
				err = vm.opCallMethod(1, "mult")
			}
		case OP_DIV:
			var ok bool
			ok, err = frame.opDiv()
			if err == nil && !ok {
				err = vm.opCallMethod(1, "div")
			}
		case OP_MOD:
			err = frame.opMod()
		case OP_CONS:
			var ok bool
			ok, err = frame.opCons()
			if err == nil && !ok {
				err = vm.opCallMethod(1, "cons")
			}
		case OP_SUB_SLICE:
			err = frame.opSubSlice()
		case OP_SUBSCRIPT_ASSIGN:
			err = frame.opSubAssign()
		case OP_LESS:
			err = frame.opLess()
		case OP_LESS_EQ:
			panic("Not implemented")
		case OP_NOT:
			err = frame.opNot()
		case OP_AND:
			err = frame.opAnd()
		case OP_OR:
			err = frame.opOr()
		case OP_SUBSCRIPT_BINARY:
			err = frame.opSubscr()
		case OP_CREATE_LIST:
			size := int(frame.readCode())
			frame.opCreateList(size)
		case OP_CREATE_MAP:
			size := int(frame.readCode())
			frame.opCreateMap(size)
		case OP_COPY:
			frame.pushStack(frame.peekStack(0))
		case OP_POP:
			frame.popStack()
		case OP_SWAP:
			// TODO: optimize
			a := frame.popStack()
			b := frame.popStack()
			frame.pushStack(a)
			frame.pushStack(b)
		case OP_JUMP:
			offset := frame.readUint16()
			frame.ip += int(offset)
		case OP_JUMP_IF_FALSE:
			offset := frame.readUint16()
			if !GetBool(frame.popStack()) {
				frame.ip += int(offset)
			}
		case OP_LOAD_GLOBAL_NAME:
			name := frame.readName()
			val, ok := vm.globals[name]
			if !ok {
				return nil, frame.runtimeError(fmt.Sprintf("Not defined: %s", name))
			}
			frame.pushStack(val)
		case OP_LOOP:
			offset := frame.readUint16()
			frame.ip -= int(offset)
		case OP_LOAD_SLOT:
			slot := uint8(frame.readCode())
			frame.pushStack(frame.stack[slot])
		case OP_LOAD_SLOT_HEAP:
			slot := uint8(frame.readCode())
			frame.pushStack(frame.stack[slot].(*BoxValue).Get())
		case OP_PUT_SLOT:
			slot := uint8(frame.readCode())
			frame.putSlot(slot, frame.popStack())
		case OP_PUT_SLOT_HEAP:
			slot := uint8(frame.readCode())
			v := frame.popStack()
			// we should never have nil pointer
			/*
				_, ok := frame.stack[slot].(*NilValue)
				// If we have a nil pointer we make a new boxed value
				if ok {
					frame.stack[slot] = &BoxValue{Nil}
				}
			*/
			frame.stack[slot].(*BoxValue).Set(v)
		case OP_PUT_GLOBAL_NAME:
			name := frame.readName()
			vm.globals[name] = frame.popStack()
		case OP_SET_METHOD:
			class := frame.readName()
			method := frame.readName()
			// TODO: make this part of the vm
			closure := frame.popStack().(*ClosureValue)
			typ, ok := vm.globals[class].(*TypeValue)
			if !ok {
				err = frame.runtimeError("Trying to define method on non-class")
			}
			typ.typ.Methods[method] = closure.Function
		case OP_CALL:
			arity := int(frame.readCode())
			vm.opCall(arity)
		case OP_CALL_METHOD:
			arity := int(frame.readCode())
			method := frame.readName()
			err = vm.opCallMethod(arity, method)
		case OP_ATTR:
			attr := frame.readName()
			err = vm.opAttr(attr)
		case OP_SET_HANDLER:
			effect := frame.readName()
			handlerIp := frame.readUint16()

			frame.pushHandler(effect, frame, int(handlerIp))
		case OP_POP_HANDLERS:
			numHandlers := int(frame.readCode())
			frame.handlers = frame.handlers[:len(frame.handlers)-numHandlers]
		case OP_DO:
			arity := int(frame.readCode())
			vm.opDo(arity)
		case OP_RESUME:
			vm.opResume()
		case OP_TYPE:
			frame.pushStack(NewTypeValue(frame.peekStack(0).Type()))
		case OP_CHECK:
			errNum := int(frame.readCode())
			err = frame.opCheck(errNum)
		default:
			return nil, fmt.Errorf("Unexpected opcode %s(%d) ", instr.String(), instr)
		}
		if err != nil {
			return nil, err
		}
	}
	// Unreachable
	return nil, nil
}

// return true if we need to call a function afterwards
func (frame *CallFrame) opEq() bool {
	b := frame.popStack()
	a := frame.popStack()

	v := builtinEq(a, b)
	if v == nil {
		frame.pushStack(a)
		frame.pushStack(b)
		return false
	}

	frame.pushStack(v)
	return true
}

// return true if we need to call a function afterwards
func (frame *CallFrame) opLess() error {
	b := frame.popStack()
	a := frame.popStack()

	switch l := a.(type) {
	case *IntValue:
		r, ok := b.(*IntValue)
		if ok {
			frame.pushStack(NewBool(l.Val < r.Val))
			return nil
		}
	default:
	}
	return frame.runtimeError(fmt.Sprintf("Trying to compare less between %s and %s", a.Type().Name, b.Type().Name))

}

func (frame *CallFrame) opNeg() error {
	a := frame.popStack()

	switch l := a.(type) {
	case *IntValue:
		frame.pushStack(NewInt(-l.Val))
	default:
		return fmt.Errorf("Trying to neg %s", a.Type().Name)
	}
	return nil
}

func (frame *CallFrame) opNot() error {
	a := frame.popStack()

	switch l := a.(type) {
	case *BoolValue:
		frame.pushStack(NewBool(!l.Val))
	default:
		return fmt.Errorf("Trying to not %s", a.Type().Name)
	}
	return nil
}

func (frame *CallFrame) opAnd() error {
	a := frame.popStack()
	b := frame.popStack()

	switch l := a.(type) {
	case *BoolValue:
		r, ok := b.(*BoolValue)
		if !ok {
			return fmt.Errorf("Trying to and %s and %s", a.Type().Name, b.Type().Name)
		} else {
			frame.pushStack(NewBool(l.Val && r.Val))
		}
	default:
		return fmt.Errorf("Trying to and %s and %s", a.Type().Name, b.Type().Name)
	}
	return nil
}

func (frame *CallFrame) opOr() error {
	a := frame.popStack()
	b := frame.popStack()

	switch l := a.(type) {
	case *BoolValue:
		r, ok := b.(*BoolValue)
		if !ok {
			return fmt.Errorf("Trying to or %s and %s", a.Type().Name, b.Type().Name)
		} else {
			frame.pushStack(NewBool(l.Val || r.Val))
		}
	default:
		return fmt.Errorf("Trying to or %s and %s", a.Type().Name, b.Type().Name)
	}
	return nil
}

func (frame *CallFrame) opSubscr() error {
	b := frame.popStack()
	a := frame.popStack()

	switch v := a.(type) {
	case *StringValue:
		r, ok := b.(*IntValue)
		if !ok {
			return fmt.Errorf("Trying to subscript %s with %s", a.Type().Name, b.Type().Name)
		} else {
			idx := r.Val
			if idx < 0 {
				idx = v.Len() - idx
			}
			val := []rune(v.Val)[idx]
			frame.pushStack(NewString(string(val)))
		}
	case *ListValue:
		r, ok := b.(*IntValue)
		if !ok {
			return fmt.Errorf("Trying to subscript %s with %s", a.Type().Name, b.Type().Name)
		} else {
			idx := r.Val
			if idx < 0 {
				idx = v.len + idx
			}
			val, ok := v.Get(idx)
			if !ok {
				return fmt.Errorf("List index out of bounds %d", r.Val)
			}
			frame.pushStack(val)
		}
	case *MapValue:
		key, ok := b.(*StringValue)
		if !ok {
			return fmt.Errorf("Trying to subscript %s with %s", a.Type().Name, b.Type().Name)
		} else {
			val, ok := v.Get(key.Val)
			if !ok {
				return fmt.Errorf("Non-existing map key: \"%s\"", key.Val)
			}
			frame.pushStack(val)
		}
	default:
		return fmt.Errorf("Trying to subscript %s with %s", a.Type().Name, b.Type().Name)
	}
	return nil
}

// return true if we could add the the operands
func (frame *CallFrame) opAdd() (bool, error) {
	res, err := builtinAdd(frame.peekStack(1), frame.peekStack(0))
	if err != nil {
		return false, err
	}
	if res != nil {
		frame.popStack()
		frame.popStack()
		frame.pushStack(res)
		return true, nil
	}
	return false, nil
}

func (frame *CallFrame) opMult() (bool, error) {
	switch l := frame.peekStack(1).(type) {
	case *IntValue:
		b := frame.popStack()
		a := frame.popStack()
		r, ok := b.(*IntValue)
		if !ok {
			return false, fmt.Errorf("Trying to mult %s and %s", a.Type().Name, b.Type().Name)
		}
		frame.pushStack(NewInt(l.Val * r.Val))
		return true, nil
	default:
		return false, nil
	}
}

func (frame *CallFrame) opSub() (bool, error) {
	switch l := frame.peekStack(1).(type) {
	case *IntValue:
		b := frame.popStack()
		a := frame.popStack()
		r, ok := b.(*IntValue)
		if !ok {
			return false, fmt.Errorf("Trying to sub %s and %s", a.Type().Name, b.Type().Name)
		} else {
			frame.pushStack(NewInt(l.Val - r.Val))
		}
	default:
		return false, nil
	}
	return true, nil
}

func (frame *CallFrame) opDiv() (bool, error) {
	switch l := frame.peekStack(1).(type) {
	case *IntValue:
		b := frame.popStack()
		a := frame.popStack()
		r, ok := b.(*IntValue)
		if !ok {
			return false, fmt.Errorf("Trying to div %s and %s", a.Type().Name, b.Type().Name)
		} else {
			frame.pushStack(NewInt(l.Val / r.Val))
		}
		return true, nil
	default:
		return false, nil
	}
}

func (frame *CallFrame) opMod() error {
	switch l := frame.peekStack(1).(type) {
	case *IntValue:
		b := frame.popStack()
		a := frame.popStack()
		r, ok := b.(*IntValue)
		if !ok {
			return frame.runtimeError(fmt.Sprintf("Trying to mod %s and %s", a.Type().Name, b.Type().Name))
		}
		frame.pushStack(NewInt(l.Val % r.Val))
		return nil
	default:
		return frame.runtimeError(fmt.Sprintf("Cant do mod on %s", l))
	}
}

func (frame *CallFrame) opCons() (bool, error) {
	switch l := frame.peekStack(0).(type) {
	case *ListValue:
		frame.popStack()
		a := frame.popStack()
		frame.pushStack(ListCons(a, l))
		return true, nil
	default:
		return false, nil
	}
}

func (frame *CallFrame) opSubSlice() error {
	c := frame.popStack()
	b := frame.popStack()
	a := frame.popStack()
	x := frame.popStack()

	intOr := func(v Value, def int) int {
		switch i := v.(type) {
		case *IntValue:
			return i.Val
		case *NilValue:
			return def
		default:
			panic("Element in subslice is not integer")
		}
	}

	from := intOr(a, 0)
	step := intOr(c, 1)

	switch v := x.(type) {
	case *ListValue:
		to := intOr(b, v.Len())
		newList := v.Slice(from, to, step)
		frame.pushStack(newList)
	case *StringValue:
		to := intOr(b, v.Len())
		s := []rune(v.Val)
		if from > to {
			return frame.runtimeError(fmt.Sprintf("Slice out of bounds %#v[%d:%d]", v.Val, from, to))
		}

		newString := NewString(string(s[from:to]))
		frame.pushStack(newString)
	default:
		panic("Subslice on non-suported value")
	}
	return nil
}

func (frame *CallFrame) opSubAssign() error {
	key := frame.popStack()
	obj := frame.popStack()
	value := frame.popStack()

	switch v := obj.(type) {
	case *MapValue:
		k, ok := key.(*StringValue)
		if !ok {
			panic("Non-string key in map assignment")
		}
		v.Set(k.Val, value)
		return nil
	}
	return frame.runtimeError(fmt.Sprintf("Can't subscript assign to %v", obj))
}

func (frame *CallFrame) opCreateList(size int) {
	v := ListNil()
	for i := 0; i < size; i++ {
		v = ListCons(frame.popStack(), v)
	}
	frame.pushStack(v)
}

func (frame *CallFrame) opCreateMap(size int) {
	v := NewMap()
	for i := 0; i < size; i++ {
		value := frame.popStack()
		key, ok := frame.popStack().(*StringValue)
		if !ok {
			panic("Expected string key in map")
		}
		v.Set(key.Val, value)
	}
	frame.pushStack(v)
}

func (vm *VM) opCall(arity int) {
	frame := vm.currentFrame
	switch fn := frame.peekStack(arity).(type) {
	case *ClosureValue:
		newFrame := vm.NewFrame(fn, frame.stack[frame.stackTop-arity:frame.stackTop], frame, frame.ip)
		vm.currentFrame = newFrame
		frame.stackTop -= arity + 1
	case *BuiltinValue:
		if arity == 1 {
			f, ok := fn.Func.(func(Value) Value)
			if !ok {
				panic(fmt.Sprintf("Calling builtin function '%s' with wrong number of arguments, expected %d", fn.Name, arity))
			}
			v := frame.popStack()
			frame.popStack()
			frame.pushStack(f(v))
		} else if arity == 2 {
			f, ok := fn.Func.(func(Value, Value) Value)
			if !ok {
				panic(fmt.Sprintf("Calling builtin function '%s' with wrong number of arguments, expected %d", fn.Name, arity))
			}
			b := frame.popStack()
			a := frame.popStack()
			frame.popStack()
			frame.pushStack(f(a, b))
		} else {
			panic(fmt.Sprintf("Not implemented arity: %d", arity))
		}
	case *TypeValue:
		// Constructor
		if arity != len(fn.typ.Attributes) {
			panic(fmt.Sprintf("Calling constructor '%s' with wrong number of arguments, expected %d", fn.typ.Name, arity))
		}
		attributes := make([]Value, arity)
		for i := arity - 1; i >= 0; i-- {
			attributes[i] = frame.popStack()
		}
		frame.popStack() // pop type
		frame.pushStack(NewCustom(fn.typ, attributes))
	default:
		panic(fmt.Sprintf("Trying to call non closure and non builtin: %v", frame.peekStack(arity)))
	}
}

func (vm *VM) opCallMethod(arity int, name string) error {
	frame := vm.currentFrame
	obj := frame.peekStack(arity)

	// Special case for type values
	t, ok := obj.(*TypeValue)
	if ok {
		method, ok := t.typ.Methods[name]
		if !ok {
			methods := strings.Join(t.typ.MethodNames(), ", ")
			return frame.runtimeError(fmt.Sprintf("No such attribute: %s on type %s. Has these methods: %s", name, obj.Type().Name, methods))
		}
		closure := NewClosure(method, []*BoxValue{})
		frame.replaceStack(arity, closure)
		vm.opCall(arity)
		return nil
	}

	method, ok := obj.Type().Methods[name]
	if !ok {
		methods := strings.Join(obj.Type().MethodNames(), ", ")
		return frame.runtimeError(fmt.Sprintf("No such attribute: %s on %s. Has these methods: %s", name, obj.Type().Name, methods))
	}

	closure := NewClosure(method, []*BoxValue{})
	// Include object on stack
	newFrame := vm.NewFrame(closure, frame.stack[frame.stackTop-arity-1:frame.stackTop], frame, frame.ip)
	vm.currentFrame = newFrame
	frame.stackTop -= arity + 1
	return nil
}

func (vm *VM) opAttr(name string) error {
	frame := vm.currentFrame
	obj := frame.popStack()

	switch t := obj.(type) {
	case *TypeValue:
		// Attribute for type methods (like `List.head`)
		method, ok := t.typ.Methods[name]
		if !ok {
			return frame.runtimeError(fmt.Sprintf("No such attribute: %s on %s", name, obj.Type().Name))
		}
		closure := NewClosure(method, []*BoxValue{})
		frame.pushStack(closure)
	case *CustomValue:
		idx := 0
		for ; idx < len(t.Attributes) && t.Type().Attributes[idx] != name; idx++ {
		}
		if idx >= len(t.Attributes) {
			return frame.runtimeError(fmt.Sprintf("No such attribute: %s on %s", name, obj.Type().Name))
		}
		frame.pushStack(t.Attributes[idx])
	default:
		return frame.runtimeError(fmt.Sprintf("Cant fetch attribute from: %s on (%s)", obj, name))
	}
	return nil
}

func (frame *CallFrame) findHandler(name string) *Handler {
	for _, h := range frame.handlers {
		if h.effect == name {
			return &h
		}
	}
	return nil
}

func (vm *VM) opDo(arity int) {
	frame := vm.currentFrame
	effect := frame.popStack().(*StringValue).Val

	var handler *Handler

	handlerFrame := frame
	for handlerFrame != nil && handler == nil {
		handler = handlerFrame.findHandler(effect)
		if handler != nil {
			break
		}
		handlerFrame = handlerFrame.returnFrame
	}

	if handler == nil {
		panic(fmt.Sprintf("No handler for effect '%s'", effect))
	}

	for i := 0; i < arity; i++ {
		v := frame.popStack()
		handlerFrame.pushStack(v)
	}

	// Create continuation
	k := NewContinuation(frame)
	handlerFrame.pushStack(k)
	handlerFrame.ip = handler.ip
	vm.currentFrame = handlerFrame
}

func (frame *CallFrame) pushHandler(effect string, handlerFrame *CallFrame, ip int) {
	frame.handlers = append(frame.handlers, struct {
		effect string
		frame  *CallFrame
		ip     int
	}{effect, handlerFrame, ip})
}

func (vm *VM) opResume() {
	v := vm.currentFrame.popStack()

	switch continuation := v.(type) {
	case *ContinuationValue:
		continuation.Frame.pushStack(vm.currentFrame.popStack())
		vm.currentFrame = continuation.Frame
	default:
		panic("Expected continuation on top of stack")
	}
}

func (frame *CallFrame) opCheck(errNum int) error {
	b, ok := frame.popStack().(*BoolValue)
	if !ok {
		return fmt.Errorf("Panic: expected bool in check operation")
	}
	if !b.Val {
		return frame.runtimeError(runtimeErrorText(errNum))
	}
	return nil
}

func (frame *CallFrame) runtimeError(msg string) error {
	line := frame.closure.Function.Chunk.LineNr[frame.ip]
	return fmt.Errorf("Runtime Error on line %d: %s", line, msg)
}
