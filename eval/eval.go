package eval

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/rymdhund/wosh/ast"
	"github.com/rymdhund/wosh/builtin"
	"github.com/rymdhund/wosh/lexer"
	. "github.com/rymdhund/wosh/obj"
)

type Runner struct {
	baseEnv *Env
	ast     *ast.BlockExpr
}

func NewRunner(ast *ast.BlockExpr) *Runner {
	return &Runner{NewOuterEnv(), ast}
}

func (runner *Runner) Run() error {
	_, exn := runner.RunExpr(runner.baseEnv, runner.ast)
	if exn != NoExnVal {
		return fmt.Errorf("Execution stopped because of unhandled exception:\nStackTrace:\n%s\n%s\n", exn.GetStackTrace(), exn.Msg())
	}
	return nil
}

// RunExpr returns a pair containing either a value for the evaluated expression or an exception value
func (runner *Runner) RunExpr(env *Env, exp ast.Expr) (Object, Exception) {
	switch v := exp.(type) {
	case *ast.BlockExpr:
		var ret Object = UnitVal
		for _, expr := range v.Children {
			var exn Exception
			ret, exn = runner.RunExpr(env, expr)
			if exn != NoExnVal {
				return UnitVal, exn
			}
		}
		return ret, NoExnVal
	case *ast.AssignExpr:
		obj, exn := runner.RunExpr(env, v.Right)
		if exn != NoExnVal {
			return UnitVal, exn
		}
		env.put(v.Ident.Name, obj)
		return UnitVal, NoExnVal
	case *ast.BasicLit:
		return objectFromBasicLit(v)
	case *ast.Ident:
		return runner.RunIdentExpr(env, v)
	case *ast.OpExpr:
		return runner.RunOpExpr(env, v)
	case *ast.UnaryExpr:
		return runner.RunUnaryExpr(env, v)
	case *ast.ListExpr:
		return runner.RunListExpr(env, v)
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
			env.put(v.Ident.Name, exn)
			return ret, NoExnVal
		default:
			panic(fmt.Sprintf("This is a bug! Invalid capture modifier: '%s'", v.Mod))
		}
	case *ast.CallExpr:
		return runner.RunCallExpr(env, v)
	case *ast.ParenthExpr:
		return runner.RunExpr(env, v.Inside)
	case *ast.CommandExpr:
		return runner.RunCommandExpr(env, v)
	case *ast.FuncDefExpr:
		fnObj := FunctionObject{v}
		if v.ClassParam != nil {
			class := env.classes[v.ClassParam.Type.Name]
			if class == nil {
				panic("Couldn't find class")
			}
			class.Methods[v.Ident.Name] = &fnObj
		} else {
			env.put(v.Ident.Name, &fnObj)
		}
		return UnitVal, NoExnVal
	default:
		panic(fmt.Sprintf("Not implemented expression in runner: %+v", exp))
	}
}

func (runner *Runner) RunOpExpr(env *Env, op *ast.OpExpr) (Object, Exception) {
	o1, exn := runner.RunExpr(env, op.Left)
	if exn != NoExnVal {
		return UnitVal, exn
	}
	o2, exn := runner.RunExpr(env, op.Right)
	if exn != NoExnVal {
		return UnitVal, exn
	}
	switch op.Op {
	case "+":
		return builtin.Add(o1, o2), NoExnVal
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
	default:
		panic(fmt.Sprintf("Not implement operator '%s'", op.Op))
	}
}

func (runner *Runner) RunUnaryExpr(env *Env, op *ast.UnaryExpr) (Object, Exception) {
	o, exn := runner.RunExpr(env, op.Right)
	if exn != NoExnVal {
		return UnitVal, exn
	}
	switch op.Op {
	case "-":
		return builtin.Neg(o), NoExnVal
	default:
		panic(fmt.Sprintf("Not implement operator '%s'", op.Op))
	}
}

func (runner *Runner) RunListExpr(env *Env, lst *ast.ListExpr) (Object, Exception) {
	list := ListNil()

	for _, expr := range lst.Elems {
		o, exn := runner.RunExpr(env, expr)
		if exn != NoExnVal {
			return UnitVal, exn
		}
		list.Add(o)
	}
	return list, NoExnVal
}

func (runner *Runner) RunSubscrExpr(env *Env, sub *ast.SubscrExpr) (Object, Exception) {
	o, exn := runner.RunExpr(env, sub.Lhs)
	if exn != NoExnVal {
		return UnitVal, exn
	}

	if len(sub.Sub) == 1 {
		idx, exn := runner.RunExpr(env, sub.Sub[0])
		if exn != NoExnVal {
			return UnitVal, exn
		}

		v, ok := builtin.Get(o, idx)
		if !ok {
			return UnitVal, ExnVal("out of bounds", "", sub.Pos().Line)
		}

		return v, NoExnVal
	} else if len(sub.Sub) == 2 || len(sub.Sub) == 3 {
		var s1, s2, s3 Object
		_, ok := sub.Sub[0].(*ast.EmptyExpr)
		if ok {
			s1 = nil
		} else {
			s1, exn = runner.RunExpr(env, sub.Sub[0])
			if exn != NoExnVal {
				return UnitVal, exn
			}
		}
		_, ok = sub.Sub[1].(*ast.EmptyExpr)
		if ok {
			s2 = nil
		} else {
			s2, exn = runner.RunExpr(env, sub.Sub[1])
			if exn != NoExnVal {
				return UnitVal, exn
			}
		}

		// s3 cannot be empty if it exists
		s3 = IntVal(1)
		if len(sub.Sub) == 3 {
			s3, exn = runner.RunExpr(env, sub.Sub[2])
			if exn != NoExnVal {
				return UnitVal, exn
			}
		}

		return builtin.Slice(o, s1, s2, s3), NoExnVal
	} else {
		panic("Unexpected number of elements in subscript")
	}
}

func (runner *Runner) RunAttrExpr(env *Env, sub *ast.AttrExpr) (Object, Exception) {
	o, exn := runner.RunExpr(env, sub.Lhs)
	if exn != NoExnVal {
		return UnitVal, exn
	}
	m := o.Class().Methods[sub.Attr.Name]
	if m == nil {
		return UnitVal, ExnVal("no such attribute", sub.Attr.Name, sub.Pos().Line)
	}
	return m, NoExnVal
}

func (runner *Runner) RunCommandExpr(env *Env, cmd *ast.CommandExpr) (Object, Exception) {
	cmdObj := exec.Command(cmd.CmdParts[0], cmd.CmdParts[1:]...)

	var stdout, stderr bytes.Buffer
	cmdObj.Stdout = &stdout
	cmdObj.Stderr = &stderr
	err := cmdObj.Run()

	outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())

	env.OutPutStr(string(outStr))
	env.ErrPutStr(string(errStr))

	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return UnitVal, ExitVal(status.ExitStatus(), strings.Join(cmd.CmdParts, " "), cmd.Pos().Line)
			}
		}
		log.Fatalf("Error running command: %s", err)
	}

	return UnitVal, NoExnVal
}

func (runner *Runner) RunCallExpr(env *Env, call *ast.CallExpr) (Object, Exception) {
	// Simple function
	ident, ok := call.Lhs.(*ast.Ident)
	if ok {
		return runner.RunCallIdent(env, call, ident)
	}

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
}

func (runner *Runner) RunCallMethod(env *Env, call *ast.CallExpr, attr *ast.AttrExpr) (Object, Exception) {
	// evaluate main object
	o, exn := runner.RunExpr(env, attr.Lhs)
	if exn != NoExnVal {
		return UnitVal, exn
	}

	// get method
	m := o.Class().Methods[attr.Attr.Name]
	if m == nil {
		return UnitVal, ExnVal("No such method", attr.Attr.Name, attr.Pos().Line)
	}

	return m, NoExnVal
}

func (runner *Runner) RunCallIdent(env *Env, call *ast.CallExpr, ident *ast.Ident) (Object, Exception) {
	switch ident.Name {
	case "echo":
		if len(call.Args) != 1 {
			panic("Expected 1 argument to echo()")
		}
		param, exn := runner.RunExpr(env, call.Args[0])
		if exn != NoExnVal {
			return UnitVal, exn
		}
		s, err := GetString(param)
		if err != nil {
			return UnitVal, ExnVal(err.Error(), ident.Name, call.Pos().Line)
		}
		env.OutPutStr(s)
		env.OutPutStr("\n")
		return UnitVal, NoExnVal
	case "echo_err":
		if len(call.Args) != 1 {
			panic("Expected 1 argument to echo_err()")
		}
		param, exn := runner.RunExpr(env, call.Args[0])
		if exn != NoExnVal {
			return UnitVal, exn
		}
		s, err := GetString(param)
		if err != nil {
			return UnitVal, ExnVal(err.Error(), ident.Name, call.Pos().Line)
		}
		env.ErrPutStr(s)
		env.ErrPutStr("\n")
		return UnitVal, NoExnVal
	case "raise":
		if len(call.Args) != 1 {
			panic("Expected 1 argument to raise()")
		}
		param, exn := runner.RunExpr(env, call.Args[0])
		// If the argument evaluation raises, we cant raise
		if exn != NoExnVal {
			return UnitVal, exn
		}
		s, err := GetString(param)
		if err != nil {
			return UnitVal, ExnVal(err.Error(), ident.Name, call.Pos().Line)
		}
		return UnitVal, ExnVal(s, "raise", call.Pos().Line)
	case "str":
		if len(call.Args) != 1 {
			panic("Expected 1 argument to str()")
		}
		param, exn := runner.RunExpr(env, call.Args[0])
		if exn != NoExnVal {
			return UnitVal, exn
		}
		s := builtin.Str(param)
		return s, NoExnVal
	case "len":
		if len(call.Args) != 1 {
			panic("Expected 1 argument to len()")
		}
		param, exn := runner.RunExpr(env, call.Args[0])
		if exn != NoExnVal {
			return UnitVal, exn
		}
		s := builtin.Len(param)
		return s, NoExnVal
	default:
		o, exn := runner.RunIdentExpr(env, ident)
		if exn != NoExnVal {
			return UnitVal, exn
		}
		return runner.RunCallObj(env, call, o, nil, ident.Name)
	}
}

func (runner *Runner) RunCallObj(env *Env, call *ast.CallExpr, o Object, classArg Object, name string) (Object, Exception) {
	f, ok := o.(*FunctionObject)
	if !ok {
		panic("cannot call non-function")
	}

	innerEnv := NewInnerEnv(env)

	if len(f.Expr.Params) != len(call.Args) {
		log.Panicf(
			"Function '%s' expected %d args, got %d",
			name,
			len(f.Expr.Params),
			len(call.Args),
		)
	}

	// Add class arg for methods
	if classArg != nil && f.Expr.ClassParam != nil {
		innerEnv.put(f.Expr.ClassParam.Name.Name, classArg)
	}

	for i, arg := range call.Args {
		param, exn := runner.RunExpr(env, arg)
		if exn != NoExnVal {
			return UnitVal, exn
		}
		innerEnv.put(f.Expr.Params[i].Name.Name, param)
	}
	res, exn := runner.RunExpr(innerEnv, f.Expr.Body)
	if exn != NoExnVal {
		exn.AddStackEntry(StackEntry{name, call.Pos().Line})
	}
	return res, exn
}

func (runner *Runner) RunIdentExpr(env *Env, ident *ast.Ident) (Object, Exception) {
	obj, ok := env.get(ident.Name)
	if !ok {
		panic(fmt.Sprintf("Undefined variable '%s'", ident.Name))
	}
	return obj, NoExnVal
}

func objectFromBasicLit(lit *ast.BasicLit) (Object, Exception) {
	switch lit.Kind {
	case lexer.INT:
		n, err := strconv.Atoi(lit.Value)
		if err != nil {
			panic(fmt.Sprintf("Expected int in basic lit: %s", err))
		}
		return IntVal(n), NoExnVal
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
	default:
		panic("Not implemented basic literal")
	}
}
