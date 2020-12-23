package interpret

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/rymdhund/wosh/ast"
	"github.com/rymdhund/wosh/lexer"
)

type Compiler struct {
	chunk            *Chunk
	localLookupTable map[string]int
	nameLookupTable  map[string]int
	arity            int
}

func (c *Compiler) getOrSetLocal(name string) uint8 {
	idx, ok := c.localLookupTable[name]
	if !ok {
		idx = len(c.chunk.LocalNames)
		c.localLookupTable[name] = idx
		c.chunk.LocalNames = append(c.chunk.LocalNames, name)
	}
	if idx > 255 {
		panic("Too many locals")
	}
	return uint8(idx)
}

func (c *Compiler) getLocalId(name string) (uint8, error) {
	idx, ok := c.localLookupTable[name]
	if !ok {
		return 0, errors.New("No such name in slots")
	}
	return uint8(idx), nil
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

func compile(function *ast.FuncDefExpr) (*FunctionValue, error) {
	arity := len(function.Params)
	if function.ClassParam != nil {
		arity++
	}
	c := Compiler{
		chunk:            NewChunk(),
		localLookupTable: map[string]int{},
		nameLookupTable:  map[string]int{},
		arity:            arity,
	}

	// On idx 0 we have function
	c.getOrSetLocal("__fn__")

	// setup the param names
	for _, param := range function.Params {
		c.getOrSetLocal(param.Name.Name)
	}

	c.CompileBlockExpr(function.Body)
	c.chunk.add(OP_RETURN, 1)
	return &FunctionValue{Name: function.Ident.Name, Arity: c.arity, Chunk: c.chunk}, nil
}

func (c *Compiler) CompileBlockExpr(block *ast.BlockExpr) error {
	for i, expr := range block.Children {
		err := c.CompileExpr(expr)
		if err != nil {
			return err
		}
		if i != len(block.Children)-1 {
			// pop the result of last expression from stack
			l := len(c.chunk.Code)
			if l > 0 && c.chunk.Code[l-1] == OP_NIL {
				// the pop cancels out the last element pushed to the stack
				c.chunk.Code = c.chunk.Code[:l-1]
			} else {
				c.chunk.add(OP_POP, expr.Pos().Line)
			}
		}
	}
	return nil
}

func (c *Compiler) CompileAssignExpr(assign *ast.AssignExpr) error {
	c.CompileExpr(assign.Right)
	slot := c.getOrSetLocal(assign.Ident.Name)
	c.chunk.add(OP_PUT_SLOT, assign.Pos().Line)
	c.chunk.add(Op(slot), assign.Pos().Line)
	c.chunk.add(OP_NIL, assign.Pos().Line)
	// doesn't put anything on stack
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
	/*
		case *ast.UnaryExpr:
			return runner.RunUnaryExpr(env, v)
		case *ast.ListExpr:
			return runner.RunListExpr(env, v)
		case *ast.MapExpr:
			return runner.RunMapExpr(env, v)
		case *ast.SubscrExpr:
			return runner.RunSubscrExpr(env, v)
		case *ast.AttrExpr:
			return runner.RunAttrExpr(env, v)
		case *ast.IfExpr:
			cond, exn := runner.RunExpr(env, v.Cond)
			if exn != NoExnVal {
				return UnitVal, exn
			}
			if GetBool(cond) {
				return runner.RunExpr(env, v.Then)
			} else if v.Else != nil {
				return runner.RunExpr(env, v.Else)
			} else {
				return UnitVal, NoExnVal
			}
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
		case *ast.ParenthExpr:
			return runner.RunExpr(env, v.Inside)
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
		constant := c.chunk.addConst(NewInt(n))
		c.chunk.add(OP_LOAD_CONSTANT, lit.Pos().Line)
		c.chunk.add(constant, lit.Pos().Line)
		/*
			case lexer.UNIT:
				return UnitVal, NoExnVal
			case lexer.BOOL:
				if lit.Value == "true" {
					return BoolVal(true), NoExnVal
				} else if lit.Value == "false" {
					return BoolVal(false), NoExnVal
				} else {
					panic(fmt.Sprintf("Expected bool in basic lit: %s", lit.Value))
				}
			case lexer.STRING:
				s := lit.Value[1 : len(lit.Value)-1]
				return StrVal(s), NoExnVal
		*/
	default:
		panic("Not implemented basic literal")
	}
	return nil
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
		c.chunk.add(OP_ADD, op.Pos().Line)
		/*
			case "-":
				return builtin.Sub(o1, o2), NoExnVal
			case "*":
				return builtin.Mult(o1, o2), NoExnVal
			case "/":
				return builtin.Div(o1, o2), NoExnVal
			case "==":
				return builtin.Eq(o1, o2), NoExnVal
			case "!=":
				return builtin.Neq(o1, o2), NoExnVal
			case "<=":
				return builtin.LessEq(o1, o2), NoExnVal
			case "<":
				return builtin.Less(o1, o2), NoExnVal
			case ">=":
				return builtin.GreaterEq(o1, o2), NoExnVal
			case ">":
				return builtin.Greater(o1, o2), NoExnVal
			case "&&":
				return builtin.BoolAnd(o1, o2), NoExnVal
			case "||":
				return builtin.BoolOr(o1, o2), NoExnVal
			case "::":
				return builtin.Cons(o1, o2), NoExnVal
		*/
	default:
		panic(fmt.Sprintf("Not implement operator '%s'", op.Op))
	}
	return nil
}

func (c *Compiler) CompileIdent(ident *ast.Ident) error {
	loc, err := c.getLocalId(ident.Name)
	if err == nil {
		// local variable
		c.chunk.add(OP_LOAD_SLOT, ident.Pos().Line)
		c.chunk.add(Op(loc), ident.Pos().Line)
	} else {
		// global variable
		nameIdx := c.getOrSetName(ident.Name)
		c.chunk.add(OP_LOAD_NAME, ident.Pos().Line)
		c.chunk.add(Op(nameIdx), ident.Pos().Line)
	}
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
		c.chunk.add(OP_CALL, call.Pos().Line)
		c.chunk.add(Op(len(call.Args)), call.Pos().Line)
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
	fnValue, err := compile(fn)
	if err != nil {
		return err
	}

	if fn.ClassParam != nil {
		panic("not implemented")
		/*
			class := env.classes[v.ClassParam.Type.Name]
			if class == nil {
				panic("Couldn't find class")
			}
			class.Methods[v.Ident.Name] = &fnObj
		*/
	} else {
		constId := c.chunk.addConst(fnValue)
		nameId := c.getOrSetName(fn.Ident.Name)
		c.chunk.add(OP_LOAD_CLOSURE, fn.Pos().Line)
		c.chunk.add(constId, fn.Pos().Line)
		c.chunk.add(OP_PUT_GLOBAL_NAME, fn.Pos().Line)
		c.chunk.add(Op(nameId), fn.Pos().Line)
		c.chunk.add(OP_NIL, fn.Pos().Line)
	}
	return nil
}

func (c *Compiler) CompileTryExpr(try *ast.TryExpr) error {
	/*
		for _, handler := range try.HandleBlock {
			nameIdx := c.getOrSetName(handler.Pattern.Ident.Name)
			c.chunk.add(OP_SET_HANDLER, ident.Pos().Line)
			c.chunk.add(Op(nameIdx), ident.Pos().Line)

		}
	*/
	return nil
}
