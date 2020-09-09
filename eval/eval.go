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
	for _, e := range runner.ast.Children {
		runner.RunExpr(runner.baseEnv, e)
	}
}

func (runner *Runner) RunExpr(env *Env, exp ast.Expr) Object {
	switch v := exp.(type) {
	case *ast.AssignExpr:
		obj := runner.RunExpr(env, v.Right)
		fmt.Printf("assigning %s = %v", v.Ident.Name, obj)
		env.put(v.Ident.Name, obj)
		return Object{"nil", 0}
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
	default:
		panic("Not implemented expression in runner")
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
		return Object{"int", n}
	default:
		panic("Not implemented basic lit")
	}
}
