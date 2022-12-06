package parser

import (
	"testing"

	"github.com/rymdhund/wosh/ast"
	"github.com/rymdhund/wosh/lexer"
)

func TestSimpleParse(t *testing.T) {
	prog := "foo"
	p := NewParser(prog)
	exprs, _, err := p.Parse()
	if err != nil {
		t.Error(err)
	}
	ident, ok := exprs.Children[0].(*ast.Ident)
	if !ok {
		t.Errorf("expected identifier(%s), got %v", prog, exprs.Children[0])
		return
	}
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
		exprs, _, err := p.Parse()
		if err != nil {
			t.Error(err)
		}
		ident, ok := exprs.Children[0].(*ast.BasicLit)
		if !ok {
			t.Errorf("expected basic lit(%s), got %v", expKind, exprs.Children[0])
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
		"f(1)",
		"f(1, 2)",
		"f(1, 2, 'hej')",
		"f(\n1,\n2,\n'hej'\n)",
	}
	for _, prog := range tests {
		p := NewParser(prog)
		exprs, _, err := p.Parse()
		if err != nil {
			t.Error(err)
		}
		_, ok := exprs.Children[0].(*ast.CallExpr)
		if !ok {
			t.Errorf("expected call expr, got %+v", exprs.Children[0])
		}
	}
}

func TestParsePipeExpr(t *testing.T) {
	tests := []string{
		"abc | def",
		"f() | g()",
		"a 1| b",
		"a 2| b",
		"a *| b",
	}
	for _, prog := range tests {
		p := NewParser(prog)
		exprs, _, err := p.Parse()
		if err != nil {
			t.Error(err)
		}
		_, ok := exprs.Children[0].(*ast.PipeExpr)
		if !ok {
			t.Errorf("expected pipe expr, got %+v", exprs.Children[0])
		}
	}
}
func TestParseAssignExpr(t *testing.T) {
	tests := []string{
		"abc = def",
		"a = g()",
	}
	for _, prog := range tests {
		p := NewParser(prog)
		exprs, _, err := p.Parse()
		if err != nil {
			t.Error(err)
		}
		_, ok := exprs.Children[0].(*ast.AssignExpr)
		if !ok {
			t.Errorf("expected AssignExpr, got %+v", exprs.Children[0])
		}
	}
}

func TestParseCaptureExpr(t *testing.T) {
	tests := []string{
		"a <- b",
		"b <-1 c",
		"b <-2 c",
		"b <-* c",
		"b <-? c",
	}
	for _, prog := range tests {
		p := NewParser(prog)
		exprs, _, err := p.Parse()
		if err != nil {
			t.Error(err)
		}
		_, ok := exprs.Children[0].(*ast.CaptureExpr)
		if !ok {
			t.Errorf("expected CaptureExpr, got %+v", exprs.Children[0])
		}
	}
}

func TestParseBlockExpr(t *testing.T) {
	tests := []struct {
		string
		int
	}{
		{"", 0},
		{"a = 1", 1},
		{"a = 1\nb = 2", 2},
		{"a = 1\n\nb = 2", 2},
		{"a = 1\nb = 2\nfoo", 3},
		{"\n\na = 1\n\nb = 2\n\nfoo", 3},
		{"\n", 0},
		{"\n a = 2 \n a \n", 2},
	}
	for _, test := range tests {
		prog, expLen := test.string, test.int
		p := NewParser(prog)
		exprs, _, err := p.Parse()
		if err != nil {
			t.Error(err)
		}
		if len(exprs.Children) != expLen {
			t.Errorf("Expected %d children, got %d", expLen, len(exprs.Children))
		}
	}
}

func TestParseIfExpr(t *testing.T) {
	tests := []struct {
		string
	}{
		{"if 1 { 2 }"},
		{"if 1 { 2 } else { 3 }"},
		{"if 1 {\n 2 } else { 4 }"},
		{"if 1 {\n a = 2 \n a \n } else { 4 }"},
		{"if 1 \n {\n 2 \n } \n else \n { \n 4 \n }\n"},
	}
	for _, test := range tests {
		prog := test.string
		p := NewParser(prog)
		exprs, _, err := p.Parse()
		if err != nil {
			t.Error(err)
		}
		_, ok := exprs.Children[0].(*ast.IfExpr)
		if !ok {
			t.Errorf("expected IfExpr, got %+v", exprs.Children[0])
		}
	}
}

func TestParseParenthExpr(t *testing.T) {
	tests := []struct {
		string
	}{
		{"(1)"},
		// Newlines dont matter in pareth exprs
		{"(\n1\n)"},
		{"(a\n=\nb)"},
		// Newlines matter in code blocks inside parenth exprs
		{"(if x {\n a=1 \n b = a \n b } else { 3 })"},
	}
	for _, test := range tests {
		prog := test.string
		p := NewParser(prog)
		exprs, _, err := p.Parse()
		if err != nil {
			t.Error(err)
		}
		_, ok := exprs.Children[0].(*ast.ParenthExpr)
		if !ok {
			t.Errorf("expected ParenthExpr, got %+v", exprs.Children[0])
		}
	}
}

func TestParseAddExpr(t *testing.T) {
	tests := []struct {
		string
	}{
		{"1 + 2"},
		{"1 + 2 + 3"},
	}
	for _, test := range tests {
		prog := test.string
		p := NewParser(prog)
		exprs, _, err := p.Parse()
		if err != nil {
			t.Error(err)
		}
		op, ok := exprs.Children[0].(*ast.OpExpr)
		if !ok {
			t.Errorf("Expected OpExpr, got %+v", exprs.Children[0])
		}
		if op.Op != "+" {
			t.Errorf("Expected +")

		}
	}
}

func TestParseOpExpr(t *testing.T) {
	tests := []struct {
		expr   string
		expect string
	}{
		{"1 - 2", "-"},
		{"1 * 2", "*"},
		{"1 / 2", "/"},
	}
	for _, test := range tests {
		p := NewParser(test.expr)
		exprs, _, err := p.Parse()
		if err != nil {
			t.Error(err)
		}
		op, ok := exprs.Children[0].(*ast.OpExpr)
		if !ok {
			t.Errorf("Expected OpExpr, got %+v", exprs.Children[0])
		}
		if op.Op != test.expect {
			t.Errorf("Expected +")

		}
	}
}

func parseForTest(t *testing.T, prog string) *ast.BlockExpr {
	p := NewParser(prog)
	exprs, _, err := p.Parse()
	if err != nil {
		t.Error(err)
	}
	return exprs
}

func TestParseAssignAdd(t *testing.T) {
	tree := parseForTest(t, "a = 1 + 2")
	assign, ok := tree.Children[0].(*ast.AssignExpr)
	if !ok {
		t.Errorf("Expected OpExpr, got %+v", tree.Children[0])
	}
	i := assign.Left.(*ast.Ident)
	if i.Name != "a" {
		t.Errorf("Invalid name")
	}
	add, ok := assign.Right.(*ast.OpExpr)
	if add.Op != "+" {
		t.Errorf("Invalid op")
	}
}

func TestParseMiscCombinations(t *testing.T) {
	tests := []struct {
		string
	}{
		{"a <- func(1)"},
		{"a <- `echo hello`"},
	}
	for _, test := range tests {
		prog := test.string
		p := NewParser(prog)
		_, _, err := p.Parse()
		if err != nil {
			t.Error(err)
		}
	}
}

func TestParseReturn(t *testing.T) {
	tests := []string{
		"return",
		"return ()",
		"return 1 + 2",
	}
	for _, prog := range tests {
		p := NewParser(prog)
		exprs, _, err := p.Parse()
		if err != nil {
			t.Error(err)
		}
		_, ok := exprs.Children[0].(*ast.ReturnExpr)
		if !ok {
			t.Errorf("expected return expr, got %+v", exprs.Children[0])
		}
	}
}

func TestParseMisc(t *testing.T) {
	tests := []string{
		"a['a'] = 1",
		"a[1] = 1",
	}
	for _, prog := range tests {
		p := NewParser(prog)
		exprs, _, err := p.Parse()
		if err != nil {
			t.Error(err)
		}
		_, ok := exprs.Children[0].(*ast.AssignExpr)
		if !ok {
			t.Errorf("expected AssignExpr, got %+v", exprs.Children[0])
		}
	}
}
