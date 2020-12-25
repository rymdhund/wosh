package interpret

import (
	"testing"

	"github.com/rymdhund/wosh/ast"
	"github.com/rymdhund/wosh/lexer"
	"github.com/rymdhund/wosh/parser"
)

func parseMain(prog string) (*ast.FuncDefExpr, error) {
	p := parser.NewParser(prog)
	exprs, err := p.Parse()
	if err != nil {
		return nil, err
	}
	return &ast.FuncDefExpr{
		Ident:      &ast.Ident{lexer.Position{0, 0}, "main"},
		ClassParam: nil,
		Params:     []*ast.ParamExpr{},
		Body:       exprs,
		TPos:       lexer.Position{0, 0},
	}, nil
}

func run(t *testing.T, prog string) Value {
	main, err := parseMain(prog)
	if err != nil {
		t.Fatal(err)
	}
	function, err := compile(main)
	if err != nil {
		t.Fatal(err)
	}
	function.DebugPrint()

	vm := NewVm()
	v, err := vm.interpret(function)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestCompile(t *testing.T) {
	prog := "x = 1"
	main, err := parseMain(prog)
	if err != nil {
		t.Fatal(err)
	}
	function, err := compile(main)
	if err != nil {
		t.Fatal(err)
	}
	function.DebugPrint()

	vm := NewVm()
	vm.interpret(function)
}

func TestAdd(t *testing.T) {
	prog := "x = 1 + 1 \n x"
	main, err := parseMain(prog)
	if err != nil {
		t.Fatal(err)
	}
	function, err := compile(main)
	if err != nil {
		t.Fatal(err)
	}
	function.DebugPrint()

	vm := NewVm()
	v, err := vm.interpret(function)
	if err != nil {
		t.Fatal(err)
	}
	if !Equal(NewInt(2), v) {
		t.Error("Expected 2")
	}
}

func TestFnDef(t *testing.T) {
	prog := "fn f() {}"
	main, err := parseMain(prog)
	if err != nil {
		t.Fatal(err)
	}
	function, err := compile(main)
	if err != nil {
		t.Fatal(err)
	}
	function.DebugPrint()

	vm := NewVm()
	v, err := vm.interpret(function)
	if err != nil {
		t.Fatal(err)
	}
	if !Equal(v, Nil) {
		t.Errorf("Expected Nil, got %s", v)
	}
}

func TestCall(t *testing.T) {
	res := run(t, "fn x(y) { y + 1} \n x(4)")
	if !Equal(res, NewInt(5)) {
		t.Errorf("expected 5, got %s", res)
	}

	res = run(t, "fn x(y) { y + 2} \n 1 + x(x(2))")
	if !Equal(res, NewInt(7)) {
		t.Errorf("expected 7, got %s", res)
	}

	res = run(t, "fn x(y) { y + 2} \n 1 + x(x(2) + 1)")
	if !Equal(res, NewInt(8)) {
		t.Errorf("expected 8, got %s", res)
	}

	res = run(t, "fn x(y) { z = y + 2 \n z + 1} \n x(1)")
	if !Equal(res, NewInt(4)) {
		t.Errorf("expected 4, got %s", res)
	}
}

func TestClosure(t *testing.T) {
	res := run(t, "fn f() { a = 1\n fn g() { a }\n g} \n b = f()\n b()")
	if !Equal(res, NewInt(1)) {
		t.Errorf("expected 1, got %s", res)
	}

	res = run(t, "fn f() { a = 1\n fn g() { a = a + 1 }\n g()\n a} \n f()")
	if !Equal(res, NewInt(2)) {
		t.Errorf("expected 2, got %s", res)
	}

	res = run(t, "fn f() { a = 1\n fn g() { a = 2 }\n g()\n a} \n f()")
	if !Equal(res, NewInt(2)) {
		t.Errorf("expected 2, got %s", res)
	}

	res = run(t, `
	fn f() {
		a = 1
		fn g() {
			fn h() {
				a = 2
			}
			h
		}
		h1 = g()
		h1()
		a
	}
	f()
	`)
	if !Equal(res, NewInt(2)) {
		t.Errorf("expected 2, got %s", res)
	}

	res = run(t, `
	fn f(x) {
		fn g(y) {
			x + y
		}
		g
	}
	add1 = f(1)
	add3 = f(3)
	add1(2) + add3(4)
	`)
	if !Equal(res, NewInt(10)) {
		t.Errorf("expected 10, got %s", res)
	}
}

func TestTry(t *testing.T) {
	res := run(t, "try { do yield(1) } handle { yield(x) -> { y = x } } \n y")
	if !Equal(res, NewInt(1)) {
		t.Errorf("expected 1, got %s", res)
	}
}
