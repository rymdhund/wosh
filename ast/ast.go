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
func (t *Ident) exprType()           {}

type BasicLit struct {
	TPos  lexer.Position
	Kind  lexer.Token
	Value string
}

func (t *BasicLit) Pos() lexer.Position { return t.TPos }
func (t *BasicLit) exprType()           {}

type Bad struct {
	TPos lexer.Position
}

func (t *Bad) Pos() lexer.Position { return t.TPos }
func (t *Bad) exprType()           {}

type CallExpr struct {
	Lhs  Expr
	Args []Expr
}

func (t *CallExpr) Pos() lexer.Position { return t.Lhs.Pos() }
func (t *CallExpr) exprType()           {}

type AttrExpr struct {
	Lhs  Expr
	Attr *Ident
}

func (t *AttrExpr) Pos() lexer.Position { return t.Lhs.Pos() }
func (t *AttrExpr) exprType()           {}

type SubscrExpr struct {
	Lhs Expr
	Sub []Expr
}

func (t *SubscrExpr) Pos() lexer.Position { return t.Lhs.Pos() }
func (t *SubscrExpr) exprType()           {}

type PipeExpr struct {
	Left      Expr
	Right     Expr
	Modifiers string
}

func (t *PipeExpr) Pos() lexer.Position { return t.Left.Pos() }
func (t *PipeExpr) exprType()           {}

type CaptureExpr struct {
	Ident *Ident
	Right Expr
	Mod   string
}

func (t *CaptureExpr) Pos() lexer.Position { return t.Ident.Pos() }
func (t *CaptureExpr) exprType()           {}

type AssignExpr struct {
	Ident *Ident
	Right Expr
}

func (t *AssignExpr) Pos() lexer.Position { return t.Ident.Pos() }
func (t *AssignExpr) exprType()           {}

type BlockExpr struct {
	Children []Expr
	TPos     lexer.Position // need to have pos if children are empty
}

func (t *BlockExpr) Pos() lexer.Position { return t.TPos }
func (t *BlockExpr) exprType()           {}

type IfExpr struct {
	Cond Expr
	Then Expr
	Else Expr
	TPos lexer.Position // need to have pos if children are empty
}

func (t *IfExpr) Pos() lexer.Position { return t.TPos }
func (t *IfExpr) exprType()           {}

type ForExpr struct {
	Cond Expr
	Then Expr
	TPos lexer.Position // need to have pos if children are empty
}

func (t *ForExpr) Pos() lexer.Position { return t.TPos }
func (t *ForExpr) exprType()           {}

type Nop struct {
	TPos lexer.Position
}

func (t *Nop) Pos() lexer.Position { return t.TPos }
func (t *Nop) exprType()           {}

// Different from Nop in that it cant be evaluated
type EmptyExpr struct {
	TPos lexer.Position
}

func (t *EmptyExpr) Pos() lexer.Position { return t.TPos }
func (t *EmptyExpr) exprType()           {}

type ParenthExpr struct {
	Inside Expr
	TPos   lexer.Position
}

func (t *ParenthExpr) Pos() lexer.Position { return t.TPos }
func (t *ParenthExpr) exprType()           {}

type OpExpr struct {
	Left  Expr
	Right Expr
	Op    string
}

func (t *OpExpr) Pos() lexer.Position { return t.Left.Pos() }
func (t *OpExpr) exprType()           {}

type UnaryExpr struct {
	Op    string
	Right Expr
	TPos  lexer.Position
}

func (t *UnaryExpr) Pos() lexer.Position { return t.TPos }
func (t *UnaryExpr) exprType()           {}

type CommandExpr struct {
	CmdParts []string
	TPos     lexer.Position
}

func (t *CommandExpr) Pos() lexer.Position { return t.TPos }
func (t *CommandExpr) exprType()           {}

type ParamExpr struct {
	Name *Ident
	Type *Ident
}

func (t *ParamExpr) Pos() lexer.Position { return t.Name.Pos() }
func (t *ParamExpr) exprType()           {}

type FuncDefExpr struct {
	Ident      *Ident
	ClassParam *ParamExpr // might be nil
	Params     []*ParamExpr
	Body       *BlockExpr
	TPos       lexer.Position
}

func (t *FuncDefExpr) Pos() lexer.Position { return t.TPos }
func (t *FuncDefExpr) exprType()           {}

type ListExpr struct {
	Elems []Expr
	TPos  lexer.Position
}

func (t *ListExpr) Pos() lexer.Position { return t.TPos }
func (t *ListExpr) exprType()           {}

type MapEntryExpr struct {
	Key *BasicLit
	Val Expr
}

func (t *MapEntryExpr) Pos() lexer.Position { return t.Key.Pos() }
func (t *MapEntryExpr) exprType()           {}

type MapExpr struct {
	Elems []*MapEntryExpr
	TPos  lexer.Position
}

func (t *MapExpr) Pos() lexer.Position { return t.TPos }
func (t *MapExpr) exprType()           {}
