package main

import (
	"testing"

	"github.com/rymdhund/wosh/ast"
	"github.com/rymdhund/wosh/lexer"
)

func TestSimpleParse(t *testing.T) {
	p := NewParser("foo")
	tree := p.Parse()
	ident := tree.(*ast.Ident)
	if ident.Name != "foo" {
		t.Errorf("wrong name")
	}
}

func TestParseBasicLit(t *testing.T) {
	tests := []struct {
		string
		lexer.Token
	}{
		{"123", lexer.INT},
		{"0123", lexer.INT},
	}
	for _, test := range tests {
		prog, expKind := test.string, test.Token
		p := NewParser(prog)
		tree := p.Parse()
		ident, ok := tree.(*ast.BasicLit)
		if !ok {
			t.Errorf("expected basic lit(%s), got %v", expKind, tree)
			continue
		}
		if ident.Value != prog {
			t.Errorf("expected BasicLit value: %s, got %s", prog, ident.Value)
		}
		if ident.Kind != expKind {
			t.Errorf("expected BasicLit.Kind: %s, got %s", expKind, ident.Kind)
		}
	}
}

func TestParseFnCall(t *testing.T) {
	tests := []string{
		"abc()",
		"ab_c ()",
	}
	for _, prog := range tests {
		p := NewParser(prog)
		tree := p.Parse()
		_, ok := tree.(*ast.CallExpr)
		if !ok {
			t.Errorf("expected call expr, got %+v", tree)
		}
	}
}

func TestParsePipeExpr(t *testing.T) {
	tests := []string{
		"abc | def",
		"f() | g()",
	}
	for _, prog := range tests {
		p := NewParser(prog)
		tree := p.Parse()
		_, ok := tree.(*ast.PipeExpr)
		if !ok {
			t.Errorf("expected pipe expr, got %+v", tree)
		}
	}
}
