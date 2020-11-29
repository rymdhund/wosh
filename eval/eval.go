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
	"github.com/rymdhund/wosh/lexer"
)

type Runner struct {
	baseEnv *Env
	ast     *ast.BlockExpr
}

func NewRunner(ast *ast.BlockExpr) *Runner {
	return &Runner{NewEnv(), ast}
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
	case *ast.FuncExpr:
		fnObj := FunctionObject{v}
		return &fnObj, NoExnVal
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
		return add(o1, o2), NoExnVal
	case "-":
		return sub(o1, o2), NoExnVal
	case "*":
		return mult(o1, o2), NoExnVal
	case "/":
		return div(o1, o2), NoExnVal
	default:
		panic(fmt.Sprintf("Not implement operator '%s'", op.Op))
	}
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
	switch call.Ident.Name {
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
			return UnitVal, ExnVal(err.Error(), call.Ident.Name, call.Pos().Line)
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
			return UnitVal, ExnVal(err.Error(), call.Ident.Name, call.Pos().Line)
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
			return UnitVal, ExnVal(err.Error(), call.Ident.Name, call.Pos().Line)
		}
		return UnitVal, ExnVal(s, "raise", call.Pos().Line)
	default:
		obj, exn := runner.RunIdentExpr(env, call.Ident)
		if exn != NoExnVal {
			return UnitVal, exn
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
			param, exn := runner.RunExpr(env, arg)
			if exn != NoExnVal {
				return UnitVal, exn
			}
			innerEnv.put(f.Expr.Args[i], param)
		}
		res, exn := runner.RunExpr(innerEnv, f.Expr.Body)
		if exn != NoExnVal {
			exn.AddStackEntry(StackEntry{call.Ident.Name, call.Pos().Line})
		}
		return res, exn
	}
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
	case lexer.STRING:
		s := lit.Value[1 : len(lit.Value)-1]
		return StrVal(s), NoExnVal
	default:
		panic("Not implemented basic literal")
	}
}
