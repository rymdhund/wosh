package eval

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"syscall"

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

func (runner *Runner) RunExpr(env *Env, exp ast.Expr) (Object, Object) {
	switch v := exp.(type) {
	case *ast.BlockExpr:
		var ret Object = UnitVal
		for _, expr := range v.Children {
			var exn Object
			ret, exn = runner.RunExpr(env, expr)
			if exn != UnitVal {
				return UnitVal, exn
			}
		}
		return ret, UnitVal
	case *ast.AssignExpr:
		obj, exn := runner.RunExpr(env, v.Right)
		if exn != UnitVal {
			return UnitVal, exn
		}
		env.put(v.Ident.Name, obj)
		return UnitVal, UnitVal
	case *ast.BasicLit:
		return objectFromBasicLit(v)
	case *ast.Ident:
		return runner.RunIdentExpr(env, v)
	case *ast.OpExpr:
		return runner.RunOpExpr(env, v)
	case *ast.IfExpr:
		cond, exn := runner.RunExpr(env, v.Cond)
		if exn != UnitVal {
			return UnitVal, exn
		}
		if GetBool(cond) {
			return runner.RunExpr(env, v.Then)
		} else if v.Else != nil {
			return runner.RunExpr(env, v.Else)
		} else {
			return UnitVal, UnitVal
		}
	case *ast.CaptureExpr:
		switch v.Mod {
		case "", "1":
			env.SetCaptureOutput()
			ret, exn := runner.RunExpr(env, v.Right)
			// Pop output even on exceptions
			env.put(v.Ident.Name, env.PopCaptureOutput())
			if exn != UnitVal {
				return UnitVal, exn
			}
			return ret, UnitVal
		case "2":
			env.SetCaptureErr()
			ret, exn := runner.RunExpr(env, v.Right)
			// Pop output even on exceptions
			env.put(v.Ident.Name, env.PopCaptureErr())
			if exn != UnitVal {
				return UnitVal, exn
			}
			return ret, UnitVal
		case "?":
			ret, exn := runner.RunExpr(env, v.Right)
			env.put(v.Ident.Name, exn)
			return ret, UnitVal
		default:
			panic(fmt.Sprintf("This is a bug! Invalid capture modifier: '%s'", v.Mod))
		}
	case *ast.CallExpr:
		return runner.RunCallExpr(env, v)
	case *ast.ParenthExpr:
		return runner.RunExpr(env, v.Inside)
	case *ast.CommandExpr:
		return runner.RunCommandExpr(env, v)
	case *ast.FuncExpr:
		fnObj := FunctionObject{v}
		return &fnObj, UnitVal
	default:
		panic(fmt.Sprintf("Not implemented expression in runner: %+v", exp))
	}
}

func (runner *Runner) RunOpExpr(env *Env, op *ast.OpExpr) (Object, Object) {
	switch op.Op {
	case "+":
		o1, exn := runner.RunExpr(env, op.Left)
		if exn != UnitVal {
			return UnitVal, exn
		}
		o2, exn := runner.RunExpr(env, op.Right)
		if exn != UnitVal {
			return UnitVal, exn
		}
		return add(o1, o2), UnitVal
	default:
		panic(fmt.Sprintf("Not implement operator '%s'", op.Op))
	}
}

func (runner *Runner) RunCommandExpr(env *Env, cmd *ast.CommandExpr) (Object, Object) {
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
				return UnitVal, ExitVal(status.ExitStatus())
			}
		}
		log.Fatalf("Error running command: %s", err)
	}

	return UnitVal, UnitVal
}

func (runner *Runner) RunCallExpr(env *Env, call *ast.CallExpr) (Object, Object) {
	switch call.Ident.Name {
	case "echo":
		if len(call.Args) != 1 {
			panic("Expected 1 argument to echo()")
		}
		param, exn := runner.RunExpr(env, call.Args[0])
		if exn != UnitVal {
			return UnitVal, exn
		}
		env.OutAdd(param)
		env.OutPutStr("\n")
		return UnitVal, UnitVal
	case "echo_err":
		if len(call.Args) != 1 {
			panic("Expected 1 argument to echo_err()")
		}
		param, exn := runner.RunExpr(env, call.Args[0])
		if exn != UnitVal {
			return UnitVal, exn
		}
		env.ErrAdd(param)
		env.ErrPutStr("\n")
		return UnitVal, UnitVal
	case "raise":
		if len(call.Args) != 1 {
			panic("Expected 1 argument to raise()")
		}
		param, exn := runner.RunExpr(env, call.Args[0])
		// If the argument evaluation raises, we cant raise
		if exn != UnitVal {
			return UnitVal, exn
		}
		return UnitVal, param
	default:
		obj, exc := runner.RunIdentExpr(env, call.Ident)
		if exc != UnitVal {
			return UnitVal, exc
		}
		f, ok := obj.(*FunctionObject)
		if !ok {
			panic("cannot call non-function")
		}

		innerEnv := NewInnerEnv(env)

		if len(f.Expr.Args) != len(call.Args) {
			log.Panicf(
				"Function '%s' expected %d args, got %d",
				call.Ident.Name,
				len(f.Expr.Args),
				len(call.Args),
			)
		}

		for i, arg := range call.Args {
			param, exc := runner.RunExpr(env, arg)
			if exc != UnitVal {
				return UnitVal, exc
			}
			innerEnv.put(f.Expr.Args[i], param)
		}
		res, exc := runner.RunExpr(innerEnv, f.Expr.Body)
		return res, exc
	}
}

func (runner *Runner) RunIdentExpr(env *Env, ident *ast.Ident) (Object, Object) {
	obj, ok := env.get(ident.Name)
	if !ok {
		panic(fmt.Sprintf("Undefined variable '%s'", ident.Name))
	}
	return obj, UnitVal
}

func objectFromBasicLit(lit *ast.BasicLit) (Object, Object) {
	switch lit.Kind {
	case lexer.INT:
		n, err := strconv.Atoi(lit.Value)
		if err != nil {
			panic(fmt.Sprintf("Expected int in basic lit: %s", err))
		}
		return IntVal(n), UnitVal
	case lexer.STRING:
		s := lit.Value[1 : len(lit.Value)-1]
		return StrVal(s), UnitVal
	default:
		panic("Not implemented basic literal")
	}
}
