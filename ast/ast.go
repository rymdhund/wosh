package ast

import (
  "github.com/rymdhund/wosh/lexer"
)


type Node interface {
  Pos() lexer.Position
  //PosEnd() lexer.Position
}

type Expr interface {
  Node
  exprType()
}


type Ident struct {
  TPos lexer.Position
  Name string
}
func (t *Ident) Pos() lexer.Position { return t.TPos }
func (t *Ident) exprType()     {}

type BasicLit struct {
  TPos lexer.Position
  Kind lexer.Token
  Value string
}
func (t *BasicLit) Pos() lexer.Position { return t.TPos }
func (t *BasicLit) exprType()     {}

type Bad struct {
  TPos lexer.Position
}
func (t *Bad) Pos() lexer.Position { return t.TPos }
func (t *Bad) exprType()     {}

type CallExpr struct {
  Ident *Ident
  Args []Expr
}
func (t *CallExpr) Pos() lexer.Position { return t.Ident.TPos }
func (t *CallExpr) exprType()     {}

type PipeExpr struct {
  Left Expr
  Right Expr
  Modifiers string
}
func (t *PipeExpr) Pos() lexer.Position { return t.Left.Pos() }
func (t *PipeExpr) exprType()     {}
