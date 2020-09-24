package eval

import "github.com/rymdhund/wosh/ast"

type FunctionObject struct {
	Expr *ast.FuncExpr
}

func (f *FunctionObject) Type() string {
	return "func"
}
