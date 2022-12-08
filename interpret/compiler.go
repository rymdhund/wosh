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
	jumpPositions []int
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
	params := function.Params
	if function.ClassParam != nil {
		params = append([]*ast.ParamExpr{function.ClassParam}, params...)
	}
	return compileFunctionFromBlock(function.Ident.Name, params, function.Body, prev)
}

func compileFunctionFromBlock(name string, params []*ast.ParamExpr, block *ast.BlockExpr, prev *Compiler) (*FunctionValue, error) {
	if DEBUG_TRACE {
		fmt.Printf("[DEBUG COMPILER] Compiling %s\n", name)
	}
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

	if err := c.CompileBlockExpr(block); err != nil {
		return nil, err
	}
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

	if DEBUG_TRACE {
		function.DebugPrint()
	}

	for _, pos := range c.jumpPositions {
		if pos != -1 {
			panic("Non-jumped placeholder")
		}
	}

	return function, nil
}

func (c *Compiler) CompileBlockExpr(block *ast.BlockExpr) error {
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
			//	c.chunk.addOp1(OP_POP, expr.StartLine())
			//}
			c.chunk.addOp1(OP_POP, expr.GetArea().End.Line)
		}
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
	case *ast.ListExpr:
		return c.CompileListExpr(v)
	case *ast.SubscrExpr:
		return c.CompileSubSlice(v)
	case *ast.ReturnExpr:
		return c.CompileReturnExpr(v)
	case *ast.AttrExpr:
		return c.CompileAttrExpr(v)
	case *ast.MapExpr:
		return c.CompileMapExpr(v)
	/*
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
		panic(fmt.Sprintf("Not implemented expression in compiler: %+v (line %d)", exp, exp.GetArea().Start.Line))
	}
}

func (c *Compiler) CompileBasicLit(lit *ast.BasicLit) error {
	switch lit.Kind {
	case lexer.INT:
		n, err := strconv.Atoi(lit.Value)
		if err != nil {
			panic(fmt.Sprintf("Expected int in basic lit: %s", err))
		}
		c.CompileConstant(NewInt(n), lit.StartLine())
	case lexer.STRING:
		c.CompileStringLit(lit)
	case lexer.BOOL:
		if lit.Value == "true" {
			c.chunk.addOp1(OP_TRUE, lit.StartLine())
		} else if lit.Value == "false" {
			c.chunk.addOp1(OP_FALSE, lit.StartLine())
		} else {
			panic(fmt.Sprintf("Expected bool in basic lit: %s", lit.Value))
		}
	case lexer.UNIT:
		c.chunk.addOp1(OP_NIL, lit.StartLine())
	default:
		panic("Not implemented basic literal")
	}
	return nil
}

func (c *Compiler) CompileStringLit(lit *ast.BasicLit) {
	s := lit.Value[1 : len(lit.Value)-1]
	c.CompileConstant(NewString(s), lit.StartLine())
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

	// lazy
	switch op.Op {
	case "&&":
		c.chunk.addOp1(OP_COPY, op.StartLine())
		firstFalse := c.addJumpToPlaceholder(OP_JUMP_IF_FALSE, op.StartLine())
		c.chunk.addOp1(OP_POP, op.StartLine())
		err = c.CompileExpr(op.Right)
		if err != nil {
			return err
		}
		c.setPlaceholder(firstFalse, c.chunk.currentPos())
		return nil
	case "||":
		c.chunk.addOp1(OP_COPY, op.StartLine())
		c.chunk.addOp1(OP_NOT, op.StartLine())
		firstTrue := c.addJumpToPlaceholder(OP_JUMP_IF_FALSE, op.StartLine())
		c.chunk.addOp1(OP_POP, op.StartLine())
		err = c.CompileExpr(op.Right)
		if err != nil {
			return err
		}
		c.setPlaceholder(firstTrue, c.chunk.currentPos())
		return nil
	}

	err = c.CompileExpr(op.Right)
	if err != nil {
		return err
	}

	switch op.Op {
	case "+":
		c.chunk.addOp1(OP_ADD, op.StartLine())
	case "-":
		c.chunk.addOp1(OP_SUB, op.StartLine())
	case "*":
		c.chunk.addOp1(OP_MULT, op.StartLine())
	case "/":
		c.chunk.addOp1(OP_DIV, op.StartLine())
	case "==":
		c.chunk.addOp1(OP_EQ, op.StartLine())
	case "!=":
		// TODO: Optimize these comparisons to only use one opcode each
		c.chunk.addOp1(OP_EQ, op.StartLine())
		c.chunk.addOp1(OP_NOT, op.StartLine())
	case "<":
		c.chunk.addOp1(OP_LESS, op.StartLine())
	case ">":
		c.chunk.addOp1(OP_SWAP, op.StartLine())
		c.chunk.addOp1(OP_LESS, op.StartLine())
	case "<=":
		c.chunk.addOp1(OP_SWAP, op.StartLine())
		c.chunk.addOp1(OP_LESS, op.StartLine())
		c.chunk.addOp1(OP_NOT, op.StartLine())
	case ">=":
		c.chunk.addOp1(OP_LESS, op.StartLine())
		c.chunk.addOp1(OP_NOT, op.StartLine())
	case "[]":
		c.chunk.addOp1(OP_SUBSCRIPT_BINARY, op.StartLine())
	case "::":
		c.chunk.addOp1(OP_CONS, op.StartLine())
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
		c.chunk.addOp1(OP_NOT, op.StartLine())
	case "-":
		c.chunk.addOp1(OP_NEG, op.StartLine())
	default:
		panic(fmt.Sprintf("Not implement operator '%s'", op.Op))
	}
	return nil
}

func (c *Compiler) CompileListExpr(lst *ast.ListExpr) error {
	for _, elem := range lst.Elems {
		err := c.CompileExpr(elem)
		if err != nil {
			return err
		}
	}
	size := len(lst.Elems)
	if size > 255 {
		panic("Too long list")
	}

	c.chunk.addOp2(OP_CREATE_LIST, Op(uint8(size)), lst.StartLine())

	return nil
}

func (c *Compiler) CompileMapExpr(m *ast.MapExpr) error {
	for _, elem := range m.Elems {
		if elem.Key.Kind != lexer.STRING {
			panic("Unexpected map key")
		}
		c.CompileStringLit(elem.Key)
		err := c.CompileExpr(elem.Val)
		if err != nil {
			return err
		}
	}
	size := len(m.Elems)
	if size > 255 {
		panic("Too long map")
	}
	c.chunk.addOp2(OP_CREATE_MAP, Op(uint8(size)), m.StartLine())

	return nil
}

func (c *Compiler) CompileSubSlice(slice *ast.SubscrExpr) error {
	if err := c.CompileExpr(slice.Lhs); err != nil {
		return err
	}

	for _, elem := range slice.Sub {
		_, ok := elem.(*ast.EmptyExpr)
		if ok {
			c.chunk.addOp1(OP_NIL, slice.StartLine())
		} else {
			err := c.CompileExpr(elem)
			if err != nil {
				return err
			}
		}
	}
	// make sure we have three arguments
	for i := len(slice.Sub); i < 3; i++ {
		c.chunk.addOp1(OP_NIL, slice.StartLine())
	}
	c.chunk.addOp1(OP_SUB_SLICE, slice.StartLine())
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
	if err := c.CompileExpr(assign.Right); err != nil {
		return err
	}

	switch v := assign.Left.(type) {
	case *ast.Ident:
		err := c.CompileAssignIdentPart(v)
		c.chunk.addOp1(OP_NIL, v.StartLine()) // result is nil
		return err
	case *ast.OpExpr:
		if v.Op == "[]" {
			err := c.CompileAssignSubscrPart(v.Left, v.Right)
			c.chunk.addOp1(OP_NIL, v.Left.GetArea().StartLine()) // result is nil
			return err
		}
	default:
		err := c.compileDestructureAssign(assign.Left)
		c.chunk.addOp1(OP_NIL, assign.Left.GetArea().StartLine()) // result is nil
		return err

	}
	return codeError(assign, "Can't assign to expression.")
}

// Check that the top of stack has type
func (c *Compiler) macroCheckType(t *Type, line int) {
	c.chunk.addOp1(OP_TYPE, line)
	c.macroCheckEquals(NewTypeValue(ListType), TYPE_ERROR, line)
}

// Check that the top of stack equals value
func (c *Compiler) macroCheckEquals(v Value, errNum int, line int) {
	c.CompileConstant(v, line)
	c.chunk.addOp1(OP_EQ, line)
	c.chunk.addOp2(OP_CHECK, Op(errNum), line)
}

func (c *Compiler) macroPutGlobalFunction(name string, line int) {
	nameIdx := c.getOrSetName(name)
	c.chunk.addOp2(OP_LOAD_GLOBAL_NAME, Op(nameIdx), line)
}

func (c *Compiler) macroCall(arity int, line int) {
	c.chunk.addOp2(OP_CALL, Op(arity), line)
}

func (c *Compiler) compileDestructureAssign(expr ast.Expr) error {
	switch v := expr.(type) {
	case *ast.Ident:
		return c.CompileAssignIdentPart(v)
	case *ast.ListExpr:
		// Do runtime checks
		c.macroCheckType(ListType, expr.GetArea().StartLine())

		// Check correct length
		c.chunk.addOp1(OP_COPY, expr.GetArea().StartLine())
		c.macroPutGlobalFunction("len", expr.GetArea().StartLine())
		c.chunk.addOp1(OP_SWAP, expr.GetArea().StartLine())
		c.macroCall(1, expr.GetArea().StartLine())
		c.macroCheckEquals(NewInt(len(v.Elems)), DESTRUCTURE_ERROR, expr.GetArea().StartLine())

		for i, elem := range v.Elems {
			if i < len(v.Elems)-1 {
				// Copy so we keep the value for the next elem
				c.chunk.addOp1(OP_COPY, expr.GetArea().StartLine())
			}
			c.CompileConstant(NewInt(i), expr.GetArea().StartLine())
			c.chunk.addOp1(OP_SUBSCRIPT_BINARY, expr.GetArea().StartLine())
			if err := c.compileDestructureAssign(elem); err != nil {
				return err
			}
		}
		return nil
	}
	return codeError(expr, "Can't assign to expression")
}

func (c *Compiler) CompileAssignIdentPart(ident *ast.Ident) error {
	slot, ok := c.lookupLocalVar(ident.Name)
	if ok && DEBUG_TRACE {
		fmt.Printf("[COMPILER DEBUG] Found local var %s\n", ident.Name)
	}
	if !ok {
		slot, ok = c.CaptureFromOuterScope(ident.Name)
		if ok && DEBUG_TRACE {
			fmt.Printf("[COMPILER DEBUG] Found outer var %s\n", ident.Name)
		}
	}

	if !ok {
		// make a new local variable
		slot = c.getOrCreateLocalVar(ident.Name)
		if DEBUG_TRACE {
			fmt.Printf("[COMPILER DEBUG] Creating local var %s\n", ident.Name)
		}
	}

	// local variable
	isHeap, ok := c.heapLookupTable[slot]
	if ok && isHeap {
		c.chunk.addOp2(OP_PUT_SLOT_HEAP, Op(slot), ident.StartLine())
	} else {
		c.chunk.addOp2(OP_PUT_SLOT, Op(slot), ident.StartLine())
	}
	return nil
}

func (c *Compiler) CompileAssignSubscrPart(lhs, key ast.Expr) error {
	if err := c.CompileExpr(lhs); err != nil {
		return err
	}
	if err := c.CompileExpr(key); err != nil {
		return err
	}

	c.chunk.addOp1(OP_SUBSCRIPT_ASSIGN, lhs.GetArea().StartLine())
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
			c.chunk.addOp2(OP_LOAD_SLOT_HEAP, Op(slot), ident.StartLine())
		} else {
			c.chunk.addOp2(OP_LOAD_SLOT, Op(slot), ident.StartLine())
		}
		return nil
	}

	// global variable
	nameIdx := c.getOrSetName(ident.Name)
	c.chunk.addOp2(OP_LOAD_GLOBAL_NAME, Op(nameIdx), ident.StartLine())
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
		c.chunk.addOp2(OP_CALL, Op(len(call.Args)), call.StartLine())
		return nil
	}

	attr, ok := call.Lhs.(*ast.AttrExpr)
	if ok {
		// Method call
		err := c.CompileExpr(attr.Lhs)
		if err != nil {
			return err
		}

		for _, expr := range call.Args {
			err := c.CompileExpr(expr)
			if err != nil {
				return err
			}
		}
		nameId := c.getOrSetName(attr.Attr.Name)
		c.chunk.addOp3(OP_CALL_METHOD, Op(len(call.Args)), Op(nameId), call.StartLine())
		return nil
	}

	panic("not implemented first class function")

	/*
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

	constId := c.chunk.addConst(fnValue)
	nameId := c.getOrSetName(fn.Ident.Name)
	c.chunk.addOp2(OP_MAKE_CLOSURE, constId, fn.StartLine())

	if fn.ClassParam == nil {
		c.chunk.addOp2(OP_PUT_GLOBAL_NAME, Op(nameId), fn.StartLine())
	} else {
		if len(fnValue.CaptureSlots) > 0 {
			panic("No capture slots expected in method!")
		}
		classNameId := c.getOrSetName(fn.ClassParam.Type.Name)
		c.chunk.addOp3(OP_SET_METHOD, Op(classNameId), Op(nameId), fn.StartLine())
	}

	c.chunk.addOp1(OP_NIL, fn.StartLine())
	return nil
}

// Retuns an id that is used by the setPlaceholder function
func (c *Compiler) addJumpToPlaceholder(jumpOp Op, line int) int {
	c.chunk.addOp3(jumpOp, Op(0x98), Op(0x76), line)
	idx := c.chunk.currentPos() - 2
	c.jumpPositions = append(c.jumpPositions, idx)
	return len(c.jumpPositions) - 1
}

// Set a placeholderId
func (c *Compiler) setPlaceholder(placeholderId, jumpDestIdx int) {
	idx := c.jumpPositions[placeholderId]

	if idx == -1 {
		panic("Placeholder Id already used")
	}

	// jump from position after placeholder
	offset := jumpDestIdx - (idx + 2)

	if offset > (1 << 16) {
		panic("Too long jump")
	}

	if offset < 0 {
		panic("Negative jump")
	}

	if c.chunk.Code[idx] != Op(0x98) || c.chunk.Code[idx+1] != Op(0x76) {
		panic("Placeholder does not contain magic values 0x98 0x76")
	}

	c.chunk.Code[idx] = Op(uint8(offset >> 8))
	c.chunk.Code[idx+1] = Op(uint8(offset))
	c.jumpPositions[placeholderId] = -1
}

func (c *Compiler) CompileTryExpr(try *ast.TryExpr) error {
	jumpToTryStart := c.addJumpToPlaceholder(OP_JUMP, try.StartLine())

	jumpToEnds := []int{}
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
			c.chunk.addOp2(OP_PUT_SLOT_HEAP, Op(slot), handler.StartLine())
		} else {
			c.chunk.addOp2(OP_PUT_SLOT, Op(slot), handler.StartLine())
		}

		for _, param := range handler.Pattern.Params {
			slot := c.createScopedLocal(param.Name.Name)
			isHeap, ok := c.heapLookupTable[slot]
			if ok && isHeap {
				c.chunk.addOp2(OP_PUT_SLOT_HEAP, Op(slot), handler.StartLine())
			} else {
				c.chunk.addOp2(OP_PUT_SLOT, Op(slot), handler.StartLine())
			}
		}

		if err := c.CompileBlockExpr(handler.Then); err != nil {
			return err
		}

		// Maybe clean up the continuation here?

		// Jump to end
		jumpToEnds = append(jumpToEnds, c.addJumpToPlaceholder(OP_JUMP, try.StartLine()))
		c.scopeEnd()
	}

	c.setPlaceholder(jumpToTryStart, c.chunk.currentPos())

	for i, handler := range handlers {
		// Set handler
		nameIdx := c.getOrSetName(handler)
		pos1, pos2 := twoBytes(handlerStarts[i])
		c.chunk.addOp4(OP_SET_HANDLER, Op(nameIdx), Op(pos1), Op(pos2), try.StartLine())
	}

	if err := c.CompileExpr(try.TryBlock); err != nil {
		return err
	}

	c.chunk.addOp2(OP_POP_HANDLERS, Op(len(try.HandleBlock)), try.StartLine())

	endPos := c.chunk.currentPos()
	for _, endJump := range jumpToEnds {
		c.setPlaceholder(endJump, endPos)
	}

	return nil
}

func (c *Compiler) CompileDoExpr(do *ast.DoExpr) error {
	for _, expr := range do.Arguments {
		err := c.CompileExpr(expr)
		if err != nil {
			return err
		}
	}
	c.CompileConstant(NewString(do.Ident.Name), do.Ident.StartLine())
	c.chunk.addOp2(OP_DO, Op(len(do.Arguments)), do.StartLine())
	return nil
}

func (c *Compiler) CompileIfExpr(iff *ast.IfExpr) error {
	err := c.CompileExpr(iff.ElifParts[0].Cond)
	if err != nil {
		return err
	}
	// Jump to next part
	lastCondFailed := c.addJumpToPlaceholder(OP_JUMP_IF_FALSE, iff.StartLine())

	err = c.CompileExpr(iff.ElifParts[0].Then)
	if err != nil {
		return err
	}

	endJumpPlaceholders := []int{}

	for _, elif := range iff.ElifParts[1:] {
		// Jump to end if previous block ran
		endJump := c.addJumpToPlaceholder(OP_JUMP, iff.StartLine())
		endJumpPlaceholders = append(endJumpPlaceholders, endJump)

		// Jump to here if previous cond failed
		c.setPlaceholder(lastCondFailed, c.chunk.currentPos())

		err := c.CompileExpr(elif.Cond)
		if err != nil {
			return err
		}
		// Jump to next part
		lastCondFailed = c.addJumpToPlaceholder(OP_JUMP_IF_FALSE, iff.StartLine())

		err = c.CompileExpr(elif.Then)
		if err != nil {
			return err
		}
	}

	if iff.Else != nil {
		// Skip else if previous condition succeeded
		endJump := c.addJumpToPlaceholder(OP_JUMP, iff.StartLine())
		endJumpPlaceholders = append(endJumpPlaceholders, endJump)

		// Jump to here if previous cond failed
		c.setPlaceholder(lastCondFailed, c.chunk.currentPos())

		err := c.CompileExpr(iff.Else)
		if err != nil {
			return err
		}

		// This is the end
		for _, jumpPlaceholder := range endJumpPlaceholders {
			c.setPlaceholder(jumpPlaceholder, c.chunk.currentPos())
		}
	} else {
		// No else block, we return NIL from expr
		c.chunk.addOp1(OP_POP, iff.StartLine())
		c.setPlaceholder(lastCondFailed, c.chunk.currentPos())
		for _, jumpPlaceholder := range endJumpPlaceholders {
			c.setPlaceholder(jumpPlaceholder, c.chunk.currentPos())
		}
		c.chunk.addOp1(OP_NIL, iff.StartLine())
	}
	return nil
}

func (c *Compiler) CompileForExpr(forr *ast.ForExpr) error {
	startIdx := c.chunk.currentPos()

	err := c.CompileExpr(forr.Cond)
	if err != nil {
		return err
	}
	jumpToEnd := c.addJumpToPlaceholder(OP_JUMP_IF_FALSE, forr.StartLine())

	err = c.CompileExpr(forr.Then)
	if err != nil {
		return err
	}
	c.chunk.addOp1(OP_POP, forr.StartLine())

	jump1, jump2 := twoBytes(c.chunk.currentPos() + 3 - startIdx)
	c.chunk.addOp3(OP_LOOP, Op(jump1), Op(jump2), forr.StartLine())
	c.setPlaceholder(jumpToEnd, c.chunk.currentPos())

	c.chunk.addOp1(OP_NIL, forr.StartLine())

	return nil
}

func (c *Compiler) CompileResumeExpr(resume *ast.ResumeExpr) error {
	if resume.Value == nil {
		c.chunk.addOp1(OP_NIL, resume.StartLine())
	} else {
		err := c.CompileExpr(resume.Value)
		if err != nil {
			return err
		}
	}
	if err := c.CompileIdent(resume.Ident); err != nil {
		return err
	}
	c.chunk.addOp1(OP_RESUME, resume.StartLine())

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

func (c *Compiler) CompileReturnExpr(ret *ast.ReturnExpr) error {
	if ret.Value == nil {
		c.chunk.addOp1(OP_RETURN_NIL, ret.StartLine())
		return nil
	}

	err := c.CompileExpr(ret.Value)
	if err != nil {
		return err
	}
	c.chunk.addOp1(OP_RETURN, ret.StartLine())

	return nil
}

func (c *Compiler) CompileAttrExpr(attr *ast.AttrExpr) error {
	err := c.CompileExpr(attr.Lhs)
	if err != nil {
		return err
	}

	nameId := c.getOrSetName(attr.Attr.Name)
	c.chunk.addOp2(OP_ATTR, Op(nameId), attr.StartLine())

	return nil

}

func codeError(e ast.Expr, msg string) error {
	return &ast.CodeError{msg, e.GetArea()}
}
