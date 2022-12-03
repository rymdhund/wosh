package ast

import (
	"github.com/rymdhund/wosh/lexer"
)

type Expr interface {
	GetArea() lexer.Area
}

type Ident struct {
	Name string
	lexer.Area
}

type BasicLit struct {
	Kind  lexer.Token
	Value string
	lexer.Area
}

type Bad struct {
	lexer.Area
}

type CallExpr struct {
	Lhs  Expr
	Args []Expr
	lexer.Area
}

type AttrExpr struct {
	Lhs  Expr
	Attr *Ident
	lexer.Area
}

type SubscrExpr struct {
	Lhs Expr
	Sub []Expr
	lexer.Area
}

type PipeExpr struct {
	Left      Expr
	Right     Expr
	Modifiers string
	lexer.Area
}

type CaptureExpr struct {
	Ident *Ident
	Right Expr
	Mod   string
	lexer.Area
}

type AssignExpr struct {
	Ident *Ident
	Right Expr
	lexer.Area
}

type BlockExpr struct {
	Children []Expr
	lexer.Area
}

type ElifPart struct {
	Cond Expr
	Then Expr
}

type IfExpr struct {
	ElifParts []ElifPart // Should always have at least one
	Else      Expr
	lexer.Area
}

type ForExpr struct {
	Cond Expr
	Then Expr
	lexer.Area
}

type Nop struct {
	lexer.Area
}

// Different from Nop in that it cant be evaluated
type EmptyExpr struct {
	lexer.Area
}

type ParenthExpr struct {
	Inside Expr
	lexer.Area
}

type OpExpr struct {
	Left  Expr
	Right Expr
	Op    string
	lexer.Area
}

type UnaryExpr struct {
	Op    string
	Right Expr
	lexer.Area
}

type CommandExpr struct {
	CmdParts []string
	lexer.Area
}

type ParamExpr struct {
	Name *Ident
	Type *Ident
	lexer.Area
}

type FuncDefExpr struct {
	Ident      *Ident
	ClassParam *ParamExpr // might be nil
	Params     []*ParamExpr
	Body       *BlockExpr
	lexer.Area
}

type ListExpr struct {
	Elems []Expr
	lexer.Area
}

type MapEntryExpr struct {
	Key *BasicLit
	Val Expr
	lexer.Area
}

type MapExpr struct {
	Elems []*MapEntryExpr
	lexer.Area
}

type MatchCaseExpr struct {
	Pattern *PatternExpr
	Then    *BlockExpr
	lexer.Area
}

// Currently only supports foo(x, y)
type PatternExpr struct {
	Ident  *Ident
	Params []*ParamExpr
	Name   *Ident //optional
	lexer.Area
}

type TryExpr struct {
	TryBlock    *BlockExpr
	HandleBlock []*MatchCaseExpr
	lexer.Area
}

type DoExpr struct {
	Ident     *Ident
	Arguments []Expr
	lexer.Area
}

type ReturnExpr struct {
	Value Expr // optional
	lexer.Area
}

type ResumeExpr struct {
	Ident *Ident
	Value Expr // optional
	lexer.Area
}
