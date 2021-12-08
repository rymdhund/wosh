package interpret

import (
	"fmt"
	"strconv"

	"github.com/rymdhund/wosh/ast"
	"github.com/rymdhund/wosh/lexer"
)

type Compiler struct {
	chunk             *Chunk
	localLookupTables []map[string]uint8
	nameLookupTable   map[string]int
	heapLookupTable   map[uint8]bool // is the variable in a given slot index on the heap?

	// when creating the closure we put slot from old scope into the captures
	outerCaptureSlots []uint8
	// When calling the closure we put captures into the slots in the new call frame
	innerCaptureSlots []uint8

	arity     int
	prevScope *Compiler

	// indexes to placeHolders for jumps etc
	placeHolders []int
}

func (c *Compiler) lookupLocalVar(name string) (uint8, bool) {
	currentScope := len(c.localLookupTables) - 1
	for currentScope >= 0 {
		idx, ok := c.localLookupTables[currentScope][name]
		if ok {
			return idx, true
		}
		currentScope--
	}
	return 0, false
}

func (c *Compiler) scopeBegin() {
	c.localLookupTables = append(c.localLookupTables, map[string]uint8{})
}

func (c *Compiler) scopeEnd() {
	c.localLookupTables = c.localLookupTables[:len(c.localLookupTables)-1]
}

func (c *Compiler) getOrCreateLocalVar(name string) uint8 {
	idx, ok := c.lookupLocalVar(name)
	if !ok {
		newIdx := len(c.chunk.LocalNames)
		if newIdx > 255 {
			panic("Too many locals")
		}
		currentScope := len(c.localLookupTables) - 1
		c.localLookupTables[currentScope][name] = uint8(newIdx)
		c.chunk.LocalNames = append(c.chunk.LocalNames, name)
		fmt.Printf("Created local var '%s' on slot %d\n", name, newIdx)
		return uint8(newIdx)
	}
	return idx
}

// Create a local variable in current scope, even if it exists in outer scope
// Will panic if variable of same name exists in current scope
func (c *Compiler) createScopedLocal(name string) uint8 {
	currentScope := len(c.localLookupTables) - 1
	_, ok := c.localLookupTables[currentScope][name]
	if ok {
		panic(fmt.Sprintf("Cant creat local variable in scope when it already exists: '%s'", name))
	}
	idx := len(c.chunk.LocalNames)
	if idx > 255 {
		panic("Too many locals")
	}
	c.localLookupTables[currentScope][name] = uint8(idx)
	c.chunk.LocalNames = append(c.chunk.LocalNames, name)
	return uint8(idx)
}

func (c *Compiler) getOrSetName(name string) uint8 {
	idx, ok := c.nameLookupTable[name]
	if !ok {
		idx = len(c.chunk.Names)
		c.nameLookupTable[name] = idx
		c.chunk.Names = append(c.chunk.Names, name)
	}
	if idx > 255 {
		panic("Too many names")
	}
	return uint8(idx)
}

func Compile(function *ast.FuncDefExpr) (*FunctionValue, error) {
	return compileFunction(function, nil)
}

func compileFunction(function *ast.FuncDefExpr, prev *Compiler) (*FunctionValue, error) {
	return compileFunctionFromBlock(function.Ident.Name, function.Params, function.Body, prev)
}

func compileFunctionFromBlock(name string, params []*ast.ParamExpr, block *ast.BlockExpr, prev *Compiler) (*FunctionValue, error) {
	fmt.Printf("Compiling %s\n", name)
	arity := len(params)
	c := Compiler{
		chunk:             NewChunk(),
		localLookupTables: []map[string]uint8{},
		nameLookupTable:   map[string]int{},
		heapLookupTable:   map[uint8]bool{},
		arity:             arity,
		prevScope:         prev,
	}
	// create initial scope
	c.scopeBegin()

	// On slot 0 we have function
	c.getOrCreateLocalVar("__fn__")

	// setup the param names
	for _, param := range params {
		c.getOrCreateLocalVar(param.Name.Name)
	}

	c.CompileBlockExpr(block)
	c.chunk.addOp1(OP_RETURN, 1)

	// find slots that should be put on heap
	heapSlots := []uint8{}
	for k, _ := range c.heapLookupTable {
		heapSlots = append(heapSlots, k)
	}

	function := &FunctionValue{
		Name:             name,
		Arity:            c.arity,
		Chunk:            c.chunk,
		OuterCaptures:    c.outerCaptureSlots,
		CaptureSlots:     c.innerCaptureSlots,
		SlotsToPutOnHeap: heapSlots,
	}

	function.DebugPrint()

	return function, nil
}

func (c *Compiler) CompileBlockExpr(block *ast.BlockExpr) error {
	if DEBUG {
		c.chunk.addNopComment("block expr starts", block.Pos().Line)
	}
	for i, expr := range block.Children {
		err := c.CompileExpr(expr)
		if err != nil {
			return err
		}
		if i != len(block.Children)-1 {
			// pop the result of last expression from stack

			// This optimization doesn't work good with jumps
			//l := len(c.chunk.Code)
			//if l > 0 && c.chunk.Code[l-1] == OP_NIL {
			//	// the pop cancels out the last element pushed to the stack
			//	c.chunk.Code = c.chunk.Code[:l-1]
			//} else {
			//	c.chunk.addOp1(OP_POP, expr.Pos().Line)
			//}
			c.chunk.addOp1(OP_POP, expr.Pos().Line)
		}
	}
	if DEBUG {
		c.chunk.addNopComment("block expr end", block.Pos().Line)
	}
	return nil
}

func (c *Compiler) CompileExpr(exp ast.Expr) error {
	switch v := exp.(type) {
	case *ast.BlockExpr:
		return c.CompileBlockExpr(v)
	case *ast.AssignExpr:
		return c.CompileAssignExpr(v)
	case *ast.BasicLit:
		return c.CompileBasicLit(v)
	case *ast.OpExpr:
		return c.CompileOpExpr(v)
	case *ast.CallExpr:
		return c.CompileCallExpr(v)
	case *ast.Ident:
		return c.CompileIdent(v)
	case *ast.FuncDefExpr:
		return c.CompileFuncDefExpr(v)
	case *ast.TryExpr:
		return c.CompileTryExpr(v)
	case *ast.DoExpr:
		return c.CompileDoExpr(v)
	case *ast.ForExpr:
		return c.CompileForExpr(v)
	case *ast.IfExpr:
		return c.CompileIfExpr(v)
	case *ast.ResumeExpr:
		return c.CompileResumeExpr(v)
	case *ast.ParenthExpr:
		return c.CompileExpr(v.Inside)
	case *ast.UnaryExpr:
		return c.CompileUnaryExpr(v)
	/*
		case *ast.ListExpr:
			return runner.RunListExpr(env, v)
		case *ast.MapExpr:
			return runner.RunMapExpr(env, v)
		case *ast.SubscrExpr:
			return runner.RunSubscrExpr(env, v)
		case *ast.AttrExpr:
			return runner.RunAttrExpr(env, v)
		case *ast.ForExpr:
			for true {
				cond, exn := runner.RunExpr(env, v.Cond)
				if exn != NoExnVal {
					return UnitVal, exn
				}
				if GetBool(cond) {
					_, exn = runner.RunExpr(env, v.Then)
					if exn != NoExnVal {
						return UnitVal, exn
					}
				} else {
					break
				}
			}
			return UnitVal, NoExnVal
		case *ast.CaptureExpr:
			switch v.Mod {
			case "", "1":
				env.SetCaptureOutput()
				ret, exn := runner.RunExpr(env, v.Right)
				// Pop output even on exceptions
				env.put(v.Ident.Name, env.PopCaptureOutput())
				if exn != NoExnVal {
					return UnitVal, exn
				}
				return ret, NoExnVal
			case "2":
				env.SetCaptureErr()
				ret, exn := runner.RunExpr(env, v.Right)
				// Pop output even on exceptions
				env.put(v.Ident.Name, env.PopCaptureErr())
				if exn != NoExnVal {
					return UnitVal, exn
				}
				return ret, NoExnVal
			case "?":
				ret, exn := runner.RunExpr(env, v.Right)
				if exn != NoExnVal {
					env.put(v.Ident.Name, exn)
				} else {
					env.put(v.Ident.Name, UnitVal)
				}
				return ret, NoExnVal
			default:
				panic(fmt.Sprintf("This is a bug! Invalid capture modifier: '%s'", v.Mod))
			}
		case *ast.CommandExpr:
			return runner.RunCommandExpr(env, v)
	*/
	default:
		panic(fmt.Sprintf("Not implemented expression in compiler: %+v", exp))
	}
}

func (c *Compiler) CompileBasicLit(lit *ast.BasicLit) error {
	switch lit.Kind {
	case lexer.INT:
		n, err := strconv.Atoi(lit.Value)
		if err != nil {
			panic(fmt.Sprintf("Expected int in basic lit: %s", err))
		}
		c.CompileConstant(NewInt(n), lit.Pos().Line)
	case lexer.STRING:
		s := lit.Value[1 : len(lit.Value)-1]
		c.CompileConstant(NewString(s), lit.Pos().Line)
	case lexer.BOOL:
		if lit.Value == "true" {
			c.chunk.addOp1(OP_TRUE, lit.Pos().Line)
		} else if lit.Value == "false" {
			c.chunk.addOp1(OP_FALSE, lit.Pos().Line)
		} else {
			panic(fmt.Sprintf("Expected bool in basic lit: %s", lit.Value))
		}
	case lexer.UNIT:
		c.chunk.addOp1(OP_NIL, lit.Pos().Line)
	default:
		panic("Not implemented basic literal")
	}
	return nil
}

func (c *Compiler) CompileConstant(value Value, line int) {
	constantIdx := c.chunk.addConst(value)
	c.chunk.addOp2(OP_LOAD_CONSTANT, constantIdx, line)
}

func (c *Compiler) CompileOpExpr(op *ast.OpExpr) error {
	err := c.CompileExpr(op.Left)
	if err != nil {
		return err
	}
	err = c.CompileExpr(op.Right)
	if err != nil {
		return err
	}

	switch op.Op {
	case "+":
		c.chunk.addOp1(OP_ADD, op.Pos().Line)
		//	case "-":
		//		return builtin.Sub(o1, o2), NoExnVal
		//	case "*":
		//		return builtin.Mult(o1, o2), NoExnVal
		//	case "/":
		//		return builtin.Div(o1, o2), NoExnVal
	case "==":
		c.chunk.addOp1(OP_EQ, op.Pos().Line)
	case "!=":
		// TODO: Optimize these comparisons to only use one opcode each
		c.chunk.addOp1(OP_EQ, op.Pos().Line)
		c.chunk.addOp1(OP_NOT, op.Pos().Line)
	case "<":
		c.chunk.addOp1(OP_LESS, op.Pos().Line)
	case ">":
		c.chunk.addOp1(OP_SWAP, op.Pos().Line)
		c.chunk.addOp1(OP_LESS, op.Pos().Line)
	case "<=":
		c.chunk.addOp1(OP_SWAP, op.Pos().Line)
		c.chunk.addOp1(OP_LESS, op.Pos().Line)
		c.chunk.addOp1(OP_NOT, op.Pos().Line)
	case ">=":
		c.chunk.addOp1(OP_LESS, op.Pos().Line)
		c.chunk.addOp1(OP_NOT, op.Pos().Line)
	case "&&":
		c.chunk.addOp1(OP_AND, op.Pos().Line)
	case "||":
		c.chunk.addOp1(OP_OR, op.Pos().Line)
		//	case "::":
		//		return builtin.Cons(o1, o2), NoExnVal
	default:
		panic(fmt.Sprintf("Not implement operator '%s'", op.Op))
	}
	return nil
}

func (c *Compiler) CompileUnaryExpr(op *ast.UnaryExpr) error {
	err := c.CompileExpr(op.Right)
	if err != nil {
		return err
	}
	switch op.Op {
	case "!":
		c.chunk.addOp1(OP_NOT, op.Pos().Line)
		//	case "-":
		//		return builtin.Sub(o1, o2), NoExnVal
		//	case "*":
		//		return builtin.Mult(o1, o2), NoExnVal
		//	case "/":
		//		return builtin.Div(o1, o2), NoExnVal
	default:
		panic(fmt.Sprintf("Not implement operator '%s'", op.Op))
	}
	return nil
}

func (c *Compiler) CaptureFromOuterScope(name string) (uint8, bool) {
	outer := c.prevScope
	// Don't capture from outmost scope since everything is global there
	if outer == nil || outer.prevScope == nil {
		return 0, false
	}

	outerSlot, ok := outer.lookupLocalVar(name)
	if ok {
		// Move captured variables to heap
		outer.MoveToHeap(outerSlot)
	} else {
		outerSlot, ok = outer.CaptureFromOuterScope(name)
	}

	if ok {
		// We found it in some higher scope, make a local Heap variable for it
		innerSlot := c.getOrCreateLocalVar(name)
		c.outerCaptureSlots = append(c.outerCaptureSlots, outerSlot)
		c.innerCaptureSlots = append(c.innerCaptureSlots, innerSlot)
		c.heapLookupTable[innerSlot] = true

		return innerSlot, true
	}

	return 0, false
}

func (c *Compiler) CompileAssignExpr(assign *ast.AssignExpr) error {
	c.CompileExpr(assign.Right)

	fmt.Printf("compiling set %s\n", assign.Ident.Name)

	slot, ok := c.lookupLocalVar(assign.Ident.Name)
	if ok {
		println("existing local")
	}
	if !ok {
		slot, ok = c.CaptureFromOuterScope(assign.Ident.Name)
		if ok {
			println("existing outer")
		}
	}

	if !ok {
		// make a new local variable
		println("new local")
		slot = c.getOrCreateLocalVar(assign.Ident.Name)
	}

	// local variable
	isHeap, ok := c.heapLookupTable[slot]
	if ok && isHeap {
		c.chunk.addOp2(OP_PUT_SLOT_HEAP, Op(slot), assign.Pos().Line)
	} else {
		c.chunk.addOp2(OP_PUT_SLOT, Op(slot), assign.Pos().Line)
	}

	c.chunk.addOp1(OP_NIL, assign.Pos().Line) // result is nil
	return nil
}

func (c *Compiler) CompileIdent(ident *ast.Ident) error {
	slot, ok := c.lookupLocalVar(ident.Name)
	if !ok {
		slot, ok = c.CaptureFromOuterScope(ident.Name)
	}

	if ok {
		// local / captured variable
		isHeap, ok := c.heapLookupTable[slot]
		if ok && isHeap {
			c.chunk.addOp2(OP_LOAD_SLOT_HEAP, Op(slot), ident.Pos().Line)
		} else {
			c.chunk.addOp2(OP_LOAD_SLOT, Op(slot), ident.Pos().Line)
		}
		return nil
	}

	// global variable
	nameIdx := c.getOrSetName(ident.Name)
	c.chunk.addOp2(OP_LOAD_GLOBAL_NAME, Op(nameIdx), ident.Pos().Line)
	return nil
}

func (c *Compiler) CompileCallExpr(call *ast.CallExpr) error {
	// Simple function
	ident, ok := call.Lhs.(*ast.Ident)
	if ok {
		err := c.CompileIdent(ident)
		if err != nil {
			return err
		}
		for _, expr := range call.Args {
			err := c.CompileExpr(expr)
			if err != nil {
				return err
			}
		}
		c.chunk.addOp2(OP_CALL, Op(len(call.Args)), call.Pos().Line)
		return nil
	}
	panic("Not implemented call")

	/*
		// method
		attr, ok := call.Lhs.(*ast.AttrExpr)
		if ok {
			return runner.RunCallMethod(env, call, attr)
		}

		// first class function somehow
		o, exn := runner.RunExpr(env, call.Lhs)
		if exn != NoExnVal {
			return UnitVal, exn
		}
		return runner.RunCallObj(env, call, o, nil, "<anonymous function>")
	*/
}

func (c *Compiler) CompileFuncDefExpr(fn *ast.FuncDefExpr) error {
	fnValue, err := compileFunction(fn, c)
	if err != nil {
		return err
	}

	if fn.ClassParam != nil {
		panic("not implemented")
	} else {
		constId := c.chunk.addConst(fnValue)
		nameId := c.getOrSetName(fn.Ident.Name)
		c.chunk.addOp2(OP_MAKE_CLOSURE, constId, fn.Pos().Line)
		c.chunk.addOp2(OP_PUT_GLOBAL_NAME, Op(nameId), fn.Pos().Line)
		c.chunk.addOp1(OP_NIL, fn.Pos().Line)
	}
	return nil
}

// Add placeholder that is 2 ops long. Requires that the previous two bytes are 0x98 0x76
func (c *Compiler) markJumpPlaceholder2() {
	idx := c.chunk.currentPos() - 2
	if c.chunk.Code[idx] != Op(0x98) || c.chunk.Code[idx+1] != Op(0x76) {
		panic("Placeholder does not contain magic values 0x98 0x76")
	}
	c.placeHolders = append(c.placeHolders, idx)
}

// make 2 op long relative jump from placeholder that is on top of the placeholder stack to idx
func (c *Compiler) makeRelativeJump2(jumpDestIdx int) {
	idx := c.placeHolders[len(c.placeHolders)-1]

	// jump from position after placeholder
	offset := jumpDestIdx - (idx + 2)

	if offset > (1 << 16) {
		panic("Too long jump")
	}

	if offset < 0 {
		panic("Negative jump")
	}

	fmt.Printf("orig idx %d\n", idx+2)
	fmt.Printf("jump idx %d\n", jumpDestIdx)
	fmt.Printf("jump offset %d\n", offset)

	c.chunk.Code[idx] = Op(uint8(offset >> 8))
	c.chunk.Code[idx+1] = Op(uint8(offset))
	c.placeHolders = c.placeHolders[:len(c.placeHolders)-1]
}

func (c *Compiler) CompileTryExpr(try *ast.TryExpr) error {
	c.chunk.addOp3(OP_JUMP, Op(0x98), Op(0x76), try.TPos.Line)
	c.markJumpPlaceholder2()

	handlers := []string{}
	handlerStarts := []int{}
	for _, handler := range try.HandleBlock {
		c.scopeBegin()
		handlers = append(handlers, handler.Pattern.Ident.Name)
		handlerStarts = append(handlerStarts, c.chunk.currentPos())
		contName := "__cont__"
		if handler.Pattern.Name != nil {
			contName = handler.Pattern.Name.Name
		}
		slot := c.createScopedLocal(contName)
		isHeap, ok := c.heapLookupTable[slot]
		if ok && isHeap {
			c.chunk.addOp2(OP_PUT_SLOT_HEAP, Op(slot), handler.TPos.Line)
		} else {
			c.chunk.addOp2(OP_PUT_SLOT, Op(slot), handler.TPos.Line)
		}

		for _, param := range handler.Pattern.Params {
			slot := c.createScopedLocal(param.Name.Name)
			isHeap, ok := c.heapLookupTable[slot]
			if ok && isHeap {
				c.chunk.addOp2(OP_PUT_SLOT_HEAP, Op(slot), handler.TPos.Line)
			} else {
				c.chunk.addOp2(OP_PUT_SLOT, Op(slot), handler.TPos.Line)
			}
		}

		c.CompileBlockExpr(handler.Then)

		// Maybe clean up the continuation here?

		c.chunk.addOp3(OP_JUMP, Op(0x98), Op(0x76), try.TPos.Line)
		c.markJumpPlaceholder2()
		c.scopeEnd()
	}

	tryStart := c.chunk.currentPos()

	for i, handler := range handlers {
		// Set handler
		nameIdx := c.getOrSetName(handler)
		pos1, pos2 := twoBytes(handlerStarts[i])
		c.chunk.addOp4(OP_SET_HANDLER, Op(nameIdx), Op(pos1), Op(pos2), try.TPos.Line)
	}

	c.CompileExpr(try.TryBlock)

	c.chunk.addOp2(OP_POP_HANDLERS, Op(len(try.HandleBlock)), try.Pos().Line)

	donePos := c.chunk.currentPos()
	for range try.HandleBlock {
		c.makeRelativeJump2(donePos)
	}
	c.makeRelativeJump2(tryStart)

	return nil
}

func (c *Compiler) CompileDoExpr(do *ast.DoExpr) error {
	for _, expr := range do.Arguments {
		err := c.CompileExpr(expr)
		if err != nil {
			return err
		}
	}
	c.CompileConstant(NewString(do.Ident.Name), do.Ident.Pos().Line)
	c.chunk.addOp2(OP_DO, Op(len(do.Arguments)), do.Pos().Line)
	return nil
}

func (c *Compiler) CompileIfExpr(iff *ast.IfExpr) error {
	err := c.CompileExpr(iff.Cond)
	if err != nil {
		return err
	}
	c.chunk.addOp3(OP_JUMP_IF_FALSE, Op(0x98), Op(0x76), iff.TPos.Line)
	c.markJumpPlaceholder2()
	err = c.CompileExpr(iff.Then)
	if err != nil {
		return err
	}
	fmt.Printf("after then %d\n", c.chunk.currentPos())
	if iff.Else != nil {
		c.chunk.addOp3(OP_JUMP, Op(0x98), Op(0x76), iff.TPos.Line)
		c.markJumpPlaceholder2()

		elseIdx := c.chunk.currentPos()
		err := c.CompileExpr(iff.Else)
		if err != nil {
			return err
		}

		c.makeRelativeJump2(c.chunk.currentPos())
		c.makeRelativeJump2(elseIdx)
	} else {
		c.chunk.addNopComment("foo", iff.TPos.Line)
		c.chunk.addOp1(OP_POP, iff.TPos.Line)
		c.makeRelativeJump2(c.chunk.currentPos())
		c.chunk.addOp1(OP_NIL, iff.TPos.Line)
	}
	return nil
}

func (c *Compiler) CompileForExpr(forr *ast.ForExpr) error {
	startIdx := c.chunk.currentPos()

	err := c.CompileExpr(forr.Cond)
	if err != nil {
		return err
	}
	c.chunk.addOp3(OP_JUMP_IF_FALSE, 0x98, 0x76, forr.TPos.Line)
	c.markJumpPlaceholder2()
	err = c.CompileExpr(forr.Then)
	if err != nil {
		return err
	}
	c.chunk.addOp1(OP_POP, forr.TPos.Line)

	jump1, jump2 := twoBytes(c.chunk.currentPos() + 3 - startIdx)
	c.chunk.addOp3(OP_LOOP, Op(jump1), Op(jump2), forr.TPos.Line)
	c.makeRelativeJump2(c.chunk.currentPos())

	c.chunk.addOp1(OP_NIL, forr.TPos.Line)

	return nil
}

func (c *Compiler) CompileResumeExpr(resume *ast.ResumeExpr) error {
	if resume.Value == nil {
		c.chunk.addOp1(OP_NIL, resume.TPos.Line)
	} else {
		err := c.CompileExpr(resume.Value)
		if err != nil {
			return err
		}
	}
	c.CompileIdent(resume.Ident)
	c.chunk.addOp1(OP_RESUME, resume.TPos.Line)

	return nil
}

func (c *Compiler) MoveToHeap(slotId uint8) {
	c.heapLookupTable[slotId] = true
	i := 0
	for i < len(c.chunk.Code) {
		if c.chunk.Code[i] == OP_PUT_SLOT {
			if uint8(c.chunk.Code[i+1]) == slotId {
				c.chunk.Code[i] = OP_PUT_SLOT_HEAP
			}
		}
		if c.chunk.Code[i] == OP_LOAD_SLOT {
			if uint8(c.chunk.Code[i+1]) == slotId {
				c.chunk.Code[i] = OP_LOAD_SLOT_HEAP
			}
		}
		if c.chunk.Code[i].Size() <= 0 {
			println("name")
			println(c.chunk.Code[i].String())
			panic("foo")
		}
		i += c.chunk.Code[i].Size()
	}
}

func twoBytes(n int) (uint8, uint8) {
	if n < 0 || (n&0x0000) != 0 {
		panic(fmt.Sprintf("Expected uint16, got %d", n))
	}
	return uint8((n >> 8) & 0xff), uint8(n & 0xff)
}
