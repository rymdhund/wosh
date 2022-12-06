package ast

import (
	"fmt"
	"strings"

	"github.com/rymdhund/wosh/lexer"
)

type Expr interface {
	GetArea() lexer.Area
	String() string
}

type Ident struct {
	Name string
	lexer.Area
}

func (v *Ident) String() string {
	return fmt.Sprintf("Ident(%s)", v.Name)
}

type BasicLit struct {
	Kind  lexer.Token
	Value string
	lexer.Area
}

func (v *BasicLit) String() string {
	return fmt.Sprintf("BasicLit(%s, %s)", v.Kind, v.Value)
}

type Bad struct {
	lexer.Area
}

func (v *Bad) String() string {
	return "Bad()"
}

type CallExpr struct {
	Lhs  Expr
	Args []Expr
	lexer.Area
}

func (v *CallExpr) String() string {
	b := strings.Builder{}
	b.WriteString("CallExpr\nLhs:\n")
	b.WriteString(tab(v.Lhs.String()))
	b.WriteString("\nArgs:")
	for _, a := range v.Args {
		b.WriteRune('\n')
		b.WriteString(tab(a.String()))
	}
	return b.String()
}

type AttrExpr struct {
	Lhs  Expr
	Attr *Ident
	lexer.Area
}

func (v *AttrExpr) String() string {
	b := strings.Builder{}
	b.WriteString("AttrExpr\nLhs:\n")
	b.WriteString(tab(v.Lhs.String()))
	b.WriteString("\nAttr: ")
	b.WriteString(v.Attr.String())
	return b.String()
}

type SubscrExpr struct {
	Lhs Expr
	Sub []Expr
	lexer.Area
}

func (v *SubscrExpr) String() string {
	b := strings.Builder{}
	b.WriteString("SubscrExpr\nLhs:\n")
	b.WriteString(tab(v.Lhs.String()))
	b.WriteString("\nSub:")
	for _, a := range v.Sub {
		b.WriteRune('\n')
		b.WriteString(tab(a.String()))
	}
	return b.String()
}

type PipeExpr struct {
	Left      Expr
	Right     Expr
	Modifiers string
	lexer.Area
}

func (v *PipeExpr) String() string {
	b := strings.Builder{}
	b.WriteString("PipeExprp(Modifiers: ")
	b.WriteString(v.Modifiers)
	b.WriteString(")\nLhs:\n")
	b.WriteString(tab(v.Left.String()))
	b.WriteString("\nRhs:\n")
	b.WriteString(tab(v.Right.String()))
	return b.String()
}

type CaptureExpr struct {
	Ident Expr
	Right Expr
	Mod   string
	lexer.Area
}

func (v *CaptureExpr) String() string {
	b := strings.Builder{}
	b.WriteString("CaptureExpr(Mod: ")
	b.WriteString(v.Mod)
	b.WriteString(")\nIdent:\n")
	b.WriteString(tab(v.Ident.String()))
	b.WriteString("\nRight:\n")
	b.WriteString(tab(v.Right.String()))
	return b.String()
}

type AssignExpr struct {
	Left  Expr
	Right Expr
	lexer.Area
}

func (v *AssignExpr) String() string {
	b := strings.Builder{}
	b.WriteString("AssignExpr\nLeft:\n")
	b.WriteString(tab(v.Left.String()))
	b.WriteString("\nRight:\n")
	b.WriteString(tab(v.Right.String()))
	return b.String()
}

type BlockExpr struct {
	Children []Expr
	lexer.Area
}

func (v *BlockExpr) String() string {
	b := strings.Builder{}
	b.WriteString("BlockExpr:")
	for _, a := range v.Children {
		b.WriteRune('\n')
		b.WriteString(tab(a.String()))
	}
	return b.String()
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

func (v *IfExpr) String() string {
	return "tbd"
}

type ForExpr struct {
	Cond Expr
	Then Expr
	lexer.Area
}

func (v *ForExpr) String() string {
	return "tbd"
}

type Nop struct {
	lexer.Area
}

func (v *Nop) String() string {
	return "Nop()"
}

// Different from Nop in that it cant be evaluated
type EmptyExpr struct {
	lexer.Area
}

func (v *EmptyExpr) String() string {
	return "Empty()"
}

type ParenthExpr struct {
	Inside Expr
	lexer.Area
}

func (v *ParenthExpr) String() string {
	return "tbd"
}

type OpExpr struct {
	Left  Expr
	Right Expr
	Op    string
	lexer.Area
}

func (v *OpExpr) String() string {
	b := strings.Builder{}
	b.WriteString("OpExpr(")
	b.WriteString(v.Op)
	b.WriteString(")\nLeft:\n")
	b.WriteString(tab(v.Left.String()))
	b.WriteString("\nRight:\n")
	b.WriteString(tab(v.Left.String()))
	return b.String()
}

type UnaryExpr struct {
	Op    string
	Right Expr
	lexer.Area
}

func (v *UnaryExpr) String() string {
	return "tbd"
}

type CommandExpr struct {
	CmdParts []string
	lexer.Area
}

func (v *CommandExpr) String() string {
	return "tbd"
}

type ParamExpr struct {
	Name *Ident
	Type *Ident
	lexer.Area
}

func (v *ParamExpr) String() string {
	return "tbd"
}

type FuncDefExpr struct {
	Ident      *Ident
	ClassParam *ParamExpr // might be nil
	Params     []*ParamExpr
	Body       *BlockExpr
	lexer.Area
}

func (v *FuncDefExpr) String() string {
	return "tbd"
}

type ListExpr struct {
	Elems []Expr
	lexer.Area
}

func (v *ListExpr) String() string {
	return "tbd"
}

type MapEntryExpr struct {
	Key *BasicLit
	Val Expr
	lexer.Area
}

func (v *MapEntryExpr) String() string {
	return "tbd"
}

type MapExpr struct {
	Elems []*MapEntryExpr
	lexer.Area
}

func (v *MapExpr) String() string {
	return "tbd"
}

type MatchCaseExpr struct {
	Pattern *PatternExpr
	Then    *BlockExpr
	lexer.Area
}

func (v *MatchCaseExpr) String() string {
	return "tbd"
}

// Currently only supports foo(x, y)
type PatternExpr struct {
	Ident  *Ident
	Params []*ParamExpr
	Name   *Ident //optional
	lexer.Area
}

func (v *PatternExpr) String() string {
	return "tbd"
}

type TryExpr struct {
	TryBlock    *BlockExpr
	HandleBlock []*MatchCaseExpr
	lexer.Area
}

func (v *TryExpr) String() string {
	return "tbd"
}

type DoExpr struct {
	Ident     *Ident
	Arguments []Expr
	lexer.Area
}

func (v *DoExpr) String() string {
	return "tbd"
}

type ReturnExpr struct {
	Value Expr // optional
	lexer.Area
}

func (v *ReturnExpr) String() string {
	return "tbd"
}

type ResumeExpr struct {
	Ident *Ident
	Value Expr // optional
	lexer.Area
}

func (v *ResumeExpr) String() string {
	return "tbd"
}

func tab(s string) string {
	lines := strings.Split(s, "\n")
	b := strings.Builder{}
	for i, line := range lines {
		if i > 0 {
			b.WriteRune('\n')
		}
		b.WriteString("  ")
		b.WriteString(line)
	}
	return b.String()
}
