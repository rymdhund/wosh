package interpret

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

const FRAMES_MAX = 64
const STACK_MAX = 256
const DEBUG_TRACE = true

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
	case *StringValue:
		return NewInt(len(x.Val))
	default:
		panic(fmt.Sprintf("%v does not support len()", v))
	}
}

func builtinPrintln(value Value) Value {
	println(value.String())
	return Nil
}

func builtinAtoi(value Value) Value {
	i, err := strconv.Atoi(value.(*StringValue).Val)
	if err != nil {
		panic(err)
	}
	return NewInt(i)
}

func NewVm() *VM {
	globals := map[string]Value{}
	globals["readlines"] = NewBuiltin("readlines", 1, builtinReadlines)
	globals["println"] = NewBuiltin("println", 1, builtinPrintln)
	globals["atoi"] = NewBuiltin("atoi", 1, builtinAtoi)
	globals["len"] = NewBuiltin("len", 1, builtinLen)
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

		switch instr {
		case OP_RETURN:
			retVal := frame.popStack()

			frame.stackTop -= len(frame.closure.Function.Chunk.LocalNames)
			// todo: clean up references for garbage collection

			if DEBUG_TRACE {
				if frame.stackTop != 0 {
					panic("expected empty stack")
				}
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
			doCall := frame.opEq()
			if doCall {
				vm.opCall(2)
			}
		case OP_ADD:
			doCall, err := frame.opAdd()
			if err != nil {
				return nil, err
			}
			if doCall {
				vm.opCall(2)
			}
		case OP_SUB:
			doCall, err := frame.opSub()
			if err != nil {
				return nil, err
			}
			if doCall {
				vm.opCall(2)
			}
		case OP_MULT:
			doCall, err := frame.opMult()
			if err != nil {
				return nil, err
			}
			if doCall {
				vm.opCall(2)
			}
		case OP_DIV:
			doCall, err := frame.opDiv()
			if err != nil {
				return nil, err
			}
			if doCall {
				vm.opCall(2)
			}
		case OP_CONS:
			doCall, err := frame.opCons()
			if err != nil {
				return nil, err
			}
			if doCall {
				vm.opCall(2)
			}
		case OP_SUB_SLICE:
			err := frame.opSubSlice()
			if err != nil {
				return nil, err
			}
		case OP_LESS:
			doCall, err := frame.opLess()
			if err != nil {
				return nil, err
			}
			if doCall {
				vm.opCall(2)
			}
		case OP_LESS_EQ:
			panic("Not implemented")
		case OP_NOT:
			err := frame.opNot()
			if err != nil {
				return nil, err
			}
		case OP_AND:
			err := frame.opAnd()
			if err != nil {
				return nil, err
			}
		case OP_OR:
			err := frame.opOr()
			if err != nil {
				return nil, err
			}
		case OP_SUBSCRIPT_BINARY:
			err := frame.opSubscr()
			if err != nil {
				return nil, err
			}
		case OP_CREATE_LIST:
			size := int(frame.readCode())
			frame.opCreateList(size)
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
				return nil, fmt.Errorf("Not defined: %s", name)
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
			types[class].Methods[method] = closure.Function
		case OP_CALL:
			arity := int(frame.readCode())
			vm.opCall(arity)
		case OP_CALL_METHOD:
			arity := int(frame.readCode())
			method := frame.readName()
			vm.opCallMethod(arity, method)
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
		default:
			return nil, fmt.Errorf("Unexpected opcode %d", instr)
		}
	}
	// Unreachable
	return nil, nil
}

// return true if we need to call a function afterwards
func (frame *CallFrame) opEq() bool {
	b := frame.popStack()
	a := frame.popStack()

	switch t := a.(type) {
	case *IntValue:
		i2, ok := b.(*IntValue)
		if !ok {
			frame.pushStack(NewBool(false))
		} else {
			frame.pushStack(NewBool(t.Val == i2.Val))
		}
	case *StringValue:
		s2, ok := b.(*StringValue)
		if !ok {
			frame.pushStack(NewBool(false))
		} else {
			frame.pushStack(NewBool(t.Val == s2.Val))
		}
	case *BoolValue:
		b2, ok := b.(*BoolValue)
		if !ok {
			frame.pushStack(NewBool(false))
		} else {
			frame.pushStack(NewBool(t.Val == b2.Val))
		}
	case *NilValue:
		_, ok := b.(*NilValue)
		frame.pushStack(NewBool(ok))
	default:
		method, ok := a.Type().Methods["eq"]
		if !ok {
			panic(fmt.Sprintf("No eq method on %s", a.Type().Name))
		}
		frame.pushStack(method)
		frame.pushStack(a)
		frame.pushStack(b)
		return true
	}
	return false
}

// return true if we need to call a function afterwards
func (frame *CallFrame) opLess() (bool, error) {
	b := frame.popStack()
	a := frame.popStack()

	switch l := a.(type) {
	case *IntValue:
		r, ok := b.(*IntValue)
		if !ok {
			return false, fmt.Errorf("Trying to compare less between %s and %s", a.Type().Name, b.Type().Name)
		} else {
			frame.pushStack(NewBool(l.Val < r.Val))
		}
	default:
		method, ok := a.Type().Methods["lt"]
		if !ok {
			panic(fmt.Sprintf("No lt method on %s", a.Type().Name))
		}
		frame.pushStack(method)
		frame.pushStack(a)
		frame.pushStack(b)
		return true, nil
	}
	return false, nil
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

	switch l := a.(type) {
	case *ListValue:
		r, ok := b.(*IntValue)
		if !ok {
			return fmt.Errorf("Trying to subscript %s with %s", a.Type().Name, b.Type().Name)
		} else {
			val, ok := l.Get(r.Val)
			if !ok {
				return fmt.Errorf("List index out of bounds")
			}
			frame.pushStack(val)
		}
	case *StringValue:
		r, ok := b.(*IntValue)
		if !ok {
			return fmt.Errorf("Trying to subscript %s with %s", a.Type().Name, b.Type().Name)
		} else {
			val := []rune(l.Val)[r.Val]
			frame.pushStack(NewString(string(val)))
		}
	default:
		return fmt.Errorf("Trying to subscript %s with %s", a.Type().Name, b.Type().Name)
	}
	return nil
}

// return true if we need to call a function afterwards
func (frame *CallFrame) opAdd() (bool, error) {
	b := frame.popStack()
	a := frame.popStack()

	switch l := a.(type) {
	case *IntValue:
		r, ok := b.(*IntValue)
		if !ok {
			return false, fmt.Errorf("Trying to add %s and %s", a.Type().Name, b.Type().Name)
		} else {
			frame.pushStack(NewInt(l.Val + r.Val))
		}
	case *StringValue:
		r, ok := b.(*StringValue)
		if !ok {
			return false, fmt.Errorf("Trying to add %s and %s", a.Type().Name, b.Type().Name)
		} else {
			frame.pushStack(NewString(l.Val + r.Val))
		}
	case *ListValue:
		r, ok := b.(*ListValue)
		if !ok {
			return false, fmt.Errorf("Trying to add %s and %s", a.Type().Name, b.Type().Name)
		} else {
			frame.pushStack(l.Concat(r))
		}
	default:
		frame.pushStack(a.Type().Methods["add"])
		frame.pushStack(a)
		frame.pushStack(b)
		return true, nil
	}
	return false, nil
}

func (frame *CallFrame) opMult() (bool, error) {
	b := frame.popStack()
	a := frame.popStack()

	switch l := a.(type) {
	case *IntValue:
		r, ok := b.(*IntValue)
		if !ok {
			return false, fmt.Errorf("Trying to mult %s and %s", a.Type().Name, b.Type().Name)
		} else {
			frame.pushStack(NewInt(l.Val * r.Val))
		}
	default:
		frame.pushStack(a.Type().Methods["mult"])
		frame.pushStack(a)
		frame.pushStack(b)
		return true, nil
	}
	return false, nil
}

func (frame *CallFrame) opSub() (bool, error) {
	b := frame.popStack()
	a := frame.popStack()

	switch l := a.(type) {
	case *IntValue:
		r, ok := b.(*IntValue)
		if !ok {
			return false, fmt.Errorf("Trying to sub %s and %s", a.Type().Name, b.Type().Name)
		} else {
			frame.pushStack(NewInt(l.Val - r.Val))
		}
	default:
		frame.pushStack(a.Type().Methods["sub"])
		frame.pushStack(a)
		frame.pushStack(b)
		return true, nil
	}
	return false, nil
}

func (frame *CallFrame) opDiv() (bool, error) {
	b := frame.popStack()
	a := frame.popStack()

	switch l := a.(type) {
	case *IntValue:
		r, ok := b.(*IntValue)
		if !ok {
			return false, fmt.Errorf("Trying to div %s and %s", a.Type().Name, b.Type().Name)
		} else {
			frame.pushStack(NewInt(l.Val / r.Val))
		}
	default:
		frame.pushStack(a.Type().Methods["sub"])
		frame.pushStack(a)
		frame.pushStack(b)
		return true, nil
	}
	return false, nil
}

func (frame *CallFrame) opCons() (bool, error) {
	b := frame.popStack()
	a := frame.popStack()
	switch l := b.(type) {
	case *ListValue:
		frame.pushStack(ListCons(a, l))
	default:
		fmt.Printf("WDF\n")
		method, ok := a.Type().Methods["cons"]
		if !ok {
			return false, fmt.Errorf("No cons method on %s", a.Type().Name)
		}

		// cons operator is right associative
		frame.pushStack(method)
		frame.pushStack(b)
		frame.pushStack(a)
		return true, nil
	}
	return false, nil
}

func (frame *CallFrame) opSubSlice() error {
	c := frame.popStack()
	b := frame.popStack()
	a := frame.popStack()
	x := frame.popStack()

	var from *IntValue
	var to *IntValue
	var step *IntValue

	switch i := a.(type) {
	case *IntValue:
		from = i
	case *NilValue:
		from = NewInt(0)
	default:
		panic("First element in subslice is not integer")
	}

	switch i := c.(type) {
	case *IntValue:
		step = i
	case *NilValue:
		step = NewInt(1)
	default:
		panic("Second element in subslice is not integer")
	}

	switch v := x.(type) {
	case *ListValue:
		switch i := b.(type) {
		case *IntValue:
			to = i
		case *NilValue:
			to = NewInt(v.len)
		default:
			panic("Second element in subslice is not integer")
		}
		newList := v.Slice(from, to, step)
		frame.pushStack(newList)
		fmt.Printf("slicing %s %s %s", from, to, step)
	case *StringValue:

		var to int
		switch i := b.(type) {
		case *IntValue:
			to = i.Val
		case *NilValue:
			to = v.Len()
		default:
			panic("Second element in subslice is not integer")
		}
		newString := NewString(v.Val[from.Val:to])
		frame.pushStack(newString)
	default:
		panic("Subslice on non-suported value")
	}
	return nil
}

func (frame *CallFrame) opCreateList(size int) {
	v := ListNil()
	for i := 0; i < size; i++ {
		v = ListCons(frame.popStack(), v)
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
			f := fn.Func.(func(Value) Value)
			v := frame.popStack()
			frame.popStack()
			frame.pushStack(f(v))
		} else {
			panic(fmt.Sprintf("Not implemented arity: %d", arity))
		}
	default:
		panic("Trying to call non closure and non builtin")
	}
}

func (vm *VM) opCallMethod(arity int, name string) {
	frame := vm.currentFrame
	obj := frame.peekStack(arity)

	method, ok := obj.Type().Methods[name]
	if !ok {
		panic(fmt.Sprintf("No such method: %s", name))
	}

	closure := NewClosure(method, []*BoxValue{})
	// Include object on stack
	newFrame := vm.NewFrame(closure, frame.stack[frame.stackTop-arity-1:frame.stackTop], frame, frame.ip)
	vm.currentFrame = newFrame
	frame.stackTop -= arity + 1
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

	fmt.Printf("Do")
	// Create continuation
	k := NewContinuation(frame)
	handlerFrame.pushStack(k)
	handlerFrame.ip = handler.ip
	vm.currentFrame = handlerFrame

	fmt.Printf("Effecting to %d\n", handlerFrame.ip)
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
