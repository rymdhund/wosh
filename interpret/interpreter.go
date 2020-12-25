package interpret

import (
	"fmt"
)

const FRAMES_MAX = 64
const STACK_MAX = 256
const DEBUG_TRACE = true
const DEBUG = true

type VM struct {
	frames     [FRAMES_MAX]CallFrame
	frameCount int
	globals    map[string]Value
}

type CallFrame struct {
	closure     *ClosureValue
	ip          int
	code        []Op // points to chunk code
	stack       [STACK_MAX]Value
	stackTop    int // points to next unused element in stack
	returnFrame int // which frame to return to
	resumeFrame int // which frame to resume to
}

func NewVm() *VM {
	return &VM{globals: map[string]Value{}}
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

func (vm *VM) NewFrame(cl *ClosureValue, args []Value) {
	frame := &vm.frames[vm.frameCount]
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
	frame.returnFrame = vm.frameCount - 1
	frame.resumeFrame = -1
	vm.frameCount++
}

func (vm *VM) interpret(main *FunctionValue) (Value, error) {
	vm.frameCount = 0
	vm.NewFrame(NewClosure(main, []*BoxValue{}), []Value{})
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
	frame := &vm.frames[vm.frameCount-1]
	for true {

		instr := frame.readCode()

		if DEBUG_TRACE {
			fmt.Printf("%-20s ", instr)
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

			if DEBUG {
				if frame.stackTop != 0 {
					panic("expected empty stack")
				}
			}

			vm.frameCount = frame.returnFrame + 1
			// todo: clean up memory
			if vm.frameCount == 0 {
				return retVal, nil
			} else {
				frame = &vm.frames[vm.frameCount-1]
				frame.pushStack(retVal)
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
		case OP_NIL:
			frame.pushStack(Nil)
		case OP_TRUE:
			frame.pushStack(NewBool(true))
		case OP_FALSE:
			frame.pushStack(NewBool(false))
		case OP_EQ:
			doCall := frame.opEq()
			if doCall {
				frame = &vm.frames[vm.frameCount-1]
				vm.opCall(2)
			}
		case OP_ADD:
			doCall, err := frame.opAdd()
			if err != nil {
				return nil, err
			}
			if doCall {
				vm.opCall(2)
				frame = &vm.frames[vm.frameCount-1]
			}
		case OP_POP:
			frame.popStack()
		case OP_JUMP:
			offset := frame.readUint16()
			frame.ip += int(offset)
		case OP_JUMP_IF_FALSE:
			offset := frame.readUint16()
			if GetBool(frame.peekStack(0)) {
				frame.ip += int(offset)
			}
		case OP_LOAD_NAME:
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
		case OP_CALL:
			arity := int(frame.readCode())
			vm.opCall(arity)
			// will have called a function
			frame = &vm.frames[vm.frameCount-1]
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
		frame.pushStack(a.Type().Methods["eq"])
		frame.pushStack(a)
		frame.pushStack(b)
		return true
	}
	return false
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

func (vm *VM) opCall(arity int) {
	frame := &vm.frames[vm.frameCount-1]
	fn := frame.peekStack(arity).(*ClosureValue)
	vm.NewFrame(fn, frame.stack[frame.stackTop-arity:frame.stackTop])

	frame.stackTop -= arity + 1
}
