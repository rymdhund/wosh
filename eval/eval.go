package eval

import (
	"fmt"
	"strconv"

	"github.com/rymdhund/wosh/ast"
	"github.com/rymdhund/wosh/lexer"
)

type Runner struct {
	baseEnv *Env
	ast     *ast.BlockExpr
}

func NewRunner(ast *ast.BlockExpr) *Runner {
	return &Runner{NewEnv(), ast}
}

func (runner *Runner) Run() {
	runner.RunExpr(runner.baseEnv, runner.ast)
}

func (runner *Runner) RunExpr(env *Env, exp ast.Expr) Object {
	switch v := exp.(type) {
	case *ast.BlockExpr:
		ret := UnitVal
		for _, expr := range v.Children {
			ret = runner.RunExpr(env, expr)
		}
		return ret
	case *ast.AssignExpr:
		obj := runner.RunExpr(env, v.Right)
		env.put(v.Ident.Name, obj)
		return UnitVal
	case *ast.BasicLit:
		return objectFromBasicLit(v)
	case *ast.Ident:
		obj, ok := env.get(v.Name)
		if !ok {
			panic(fmt.Sprintf("Undefined variable '%s'", v.Name))
		}
		return obj
	case *ast.OpExpr:
		return runner.RunOpExpr(env, v)
	case *ast.IfExpr:
		cond := runner.RunExpr(env, v.Cond)
		if cond.Type != "int" {
			// TODO
			panic("Not implemented boolean type")
		}
		if cond.Value != 0 {
			return runner.RunExpr(env, v.Then)
		} else if v.Else != nil {
			return runner.RunExpr(env, v.Else)
		} else {
			return UnitVal
		}
	case *ast.CaptureExpr:
		switch v.Mod {
		case "", "1":
			env.SetCaptureOutput()
			ret := runner.RunExpr(env, v.Right)
			env.put(v.Ident.Name, env.PopCaptureOutput())
			return ret
		case "2":
			env.SetCaptureErr()
			ret := runner.RunExpr(env, v.Right)
			env.put(v.Ident.Name, env.PopCaptureErr())
			return ret
		default:
			panic(fmt.Sprintf("This is a bug! Invalid capture modifier: '%s'", v.Mod))
		}
	case *ast.CallExpr:
		switch v.Ident.Name {
		case "echo":
			if len(v.Args) != 1 {
				panic("Expected 1 argument to echo()")
			}
			param := runner.RunExpr(env, v.Args[0])
			env.OutAdd(param)
			env.OutPutStr("\n")
			return UnitVal
		case "echo_err":
			if len(v.Args) != 1 {
				panic("Expected 1 argument to echo_err()")
			}
			param := runner.RunExpr(env, v.Args[0])
			env.ErrAdd(param)
			env.ErrPutStr("\n")
			return UnitVal
		default:
			panic(fmt.Sprintf("Unknown function %s", v.Ident.Name))
		}
	default:
		panic(fmt.Sprintf("Not implemented expression in runner: %+v", exp))
	}
}

func (runner *Runner) RunOpExpr(env *Env, op *ast.OpExpr) Object {
	switch op.Op {
	case "+":
		o1 := runner.RunExpr(env, op.Left)
		o2 := runner.RunExpr(env, op.Right)
		return o1.add(o2)
	default:
		panic(fmt.Sprintf("Not implement operator '%s'", op.Op))
	}
}

func objectFromBasicLit(lit *ast.BasicLit) Object {
	switch lit.Kind {
	case lexer.INT:
		n, err := strconv.Atoi(lit.Value)
		if err != nil {
			panic(fmt.Sprintf("Expected int in basic lit: %s", err))
		}
		return IntVal(n)
	case lexer.STRING:
		s := lit.Value[1 : len(lit.Value)-1]
		return StrVal(s)
	default:
		panic("Not implemented basic literal")
	}
}
