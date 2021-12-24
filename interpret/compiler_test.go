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
	function, err := Compile(main)
	if err != nil {
		t.Fatal(err)
	}

	vm := NewVm()
	v, err := vm.Interpret(function)
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
	function, err := Compile(main)
	if err != nil {
		t.Fatal(err)
	}
	function.DebugPrint()

	vm := NewVm()
	vm.Interpret(function)
}

func TestAdd(t *testing.T) {
	prog := "x = 1 + 1 \n x"
	main, err := parseMain(prog)
	if err != nil {
		t.Fatal(err)
	}
	function, err := Compile(main)
	if err != nil {
		t.Fatal(err)
	}
	function.DebugPrint()

	vm := NewVm()
	v, err := vm.Interpret(function)
	if err != nil {
		t.Fatal(err)
	}
	if !Equal(NewInt(2), v) {
		t.Error("Expected 2")
	}
}

func TestIf(t *testing.T) {
	res := run(t, "if 1 == 1 { 2 } else { 3 }")
	if !Equal(res, NewInt(2)) {
		t.Errorf("expected 2, got %s", res)
	}

	res = run(t, "if 1 == 2 { 3 } else { 4 }")
	if !Equal(res, NewInt(4)) {
		t.Errorf("expected 4, got %s", res)
	}

	res = run(t, "x = 0 \n if 1 == 1 { x = 10 } \n x")
	if !Equal(res, NewInt(10)) {
		t.Errorf("expected 10, got %s", res)
	}

	res = run(t, "x = 0 \n if 1 == 2 { x = 10 } \n x")
	if !Equal(res, NewInt(0)) {
		t.Errorf("expected 0, got %s", res)
	}
}

func TestFor(t *testing.T) {
	res := run(t, "x = 1 \n for x == 1 { x = x + 1 } \n x")
	if !Equal(res, NewInt(2)) {
		t.Errorf("expected 2, got %s", res)
	}

	res = run(t, "x = 1 \n for x < 10 { x = x + 1 } \n x")
	if !Equal(res, NewInt(10)) {
		t.Errorf("expected 10, got %s", res)
	}

	res = run(t, "x = 1 \n for false { x = x + 1 } \n x")
	if !Equal(res, NewInt(1)) {
		t.Errorf("expected 1, got %s", res)
	}
}

func TestFnDef(t *testing.T) {
	prog := "fn f() {}"
	main, err := parseMain(prog)
	if err != nil {
		t.Fatal(err)
	}
	function, err := Compile(main)
	if err != nil {
		t.Fatal(err)
	}
	function.DebugPrint()

	vm := NewVm()
	v, err := vm.Interpret(function)
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
	res := run(t, `
	fn foo() {
		y = 0
		try {
			do yield(1)
		} handle {
			yield(x) -> { y = x }
		}
		y
	}
	foo()
	`)
	if !Equal(res, NewInt(1)) {
		t.Errorf("expected 1, got %s", res)
	}
}

func TestTry2(t *testing.T) {
	res := run(t, `
	fn foo() {
		y = 0
		try {
			do yield(1)
			y = 3  # will not be evaluated
		} handle {
			yield(x) -> {
				y = x
			}
		}
		y
	}
	foo()
	`)
	if !Equal(res, NewInt(1)) {
		t.Errorf("expected 1, got %s", res)
	}
}

func TestTry3(t *testing.T) {
	res := run(t, `
	fn bar() {
		do yield(3)
	}
	fn foo() {
		y = 0
		try {
			bar()
		} handle {
			yield(x) -> {
				y = x
			}
		}
		y
	}
	foo()
	`)
	if !Equal(res, NewInt(3)) {
		t.Errorf("expected 3, got %s", res)
	}
}

func TestTry4(t *testing.T) {
	res := run(t, `
	fn bar() {
		do yield(3)
	}
	fn baz() {
		bar()
	}
	fn foo() {
		y = 0
		try {
			baz()
		} handle {
			yield(x) -> {
				y = x
			}
		}
		y
	}
	foo()
	`)
	if !Equal(res, NewInt(3)) {
		t.Errorf("expected 3, got %s", res)
	}
}

func TestTry5(t *testing.T) {
	res := run(t, `
	fn count() {
		i = 0
		for 1 == 1 {
			do yield(i)
			i = i + 1
		}
	}
	fn foo() {
		y = 0
		try {
			count()
		} handle {
			yield(x) @ k -> {
				y = x
				if x < 3 {
					resume k
				}
			}
		}
		y
	}
	foo()
	`)
	if !Equal(res, NewInt(3)) {
		t.Errorf("expected 3, got %s", res)
	}
}

func TestTry6(t *testing.T) {
	res := run(t, `
	try {
		3
	} handle {
		yield(x) -> {
			4
		}
	}
	`)
	if !Equal(res, NewInt(3)) {
		t.Errorf("expected 3, got %s", res)
	}
}

func TestTry7(t *testing.T) {
	res := run(t, `
	try {
		do yield(1)
		3
	} handle {
		yield(x) -> {
			4
		}
	}
	`)
	if !Equal(res, NewInt(4)) {
		t.Errorf("expected 4, got %s", res)
	}
}

func TestTry8(t *testing.T) {
	res := run(t, `
	fn count() {
		do eff1(100)
		do eff2(10)
		3
	}
	fn foo() {
		sum = 0
		try {
			count() + sum
		} handle {
			eff1(x) @ k -> {
				sum = sum + x
				resume k
			}
			eff2(x) @ k -> {
				sum = sum + x
				resume k
			}
		}
	}
	foo()
	`)
	if !Equal(res, NewInt(113)) {
		t.Errorf("expected 113, got %s", res)
	}
}

func assertFalse(t *testing.T, prog string) {
	res := run(t, prog)
	if !Equal(res, NewBool(false)) {
		t.Errorf("expected false, got %s", res)
	}
}

func assertTrue(t *testing.T, prog string) {
	res := run(t, prog)
	if !Equal(res, NewBool(true)) {
		t.Errorf("expected true, got %s", res)
	}
}

func assertInt(t *testing.T, prog string, value int) {
	res := run(t, prog)
	if !Equal(res, NewInt(value)) {
		t.Errorf("expected %d, got %s", value, res)
	}
}

func TestArithmetic(t *testing.T) {
	assertInt(t, "1 + 1", 2)
	assertInt(t, "1 * 1", 1)
	assertInt(t, "1 - 1", 0)
	assertInt(t, "1 / 1", 1)

	assertInt(t, "1 + 2 * 3", 7)
	assertInt(t, "2 * 3 + 4", 10)

	assertInt(t, "1 + 2 * 3 - 4 / 2", 5)
}

func TestCompare(t *testing.T) {
	assertFalse(t, "1 > 2")
	assertFalse(t, "1 > 1")
	assertTrue(t, "2 > 1")

	assertFalse(t, "1 >= 2")
	assertTrue(t, "1 >= 1")
	assertTrue(t, "2 >= 1")

	assertTrue(t, "1 < 2")
	assertFalse(t, "1 < 1")
	assertFalse(t, "2 < 1")

	assertTrue(t, "1 <= 2")
	assertTrue(t, "1 <= 1")
	assertFalse(t, "2 <= 1")

	assertTrue(t, "1 == 1")
	assertFalse(t, "1 == 2")
	assertFalse(t, "2 == 1")

	assertFalse(t, "1 != 1")
	assertTrue(t, "1 != 2")
	assertTrue(t, "2 != 1")
}

func TestBool(t *testing.T) {
	assertTrue(t, "true")
	assertFalse(t, "false")

	assertFalse(t, "!true")
	assertTrue(t, "!false")

	assertTrue(t, "true || true")
	assertTrue(t, "true || false")
	assertTrue(t, "false || true")
	assertFalse(t, "false || false")

	assertTrue(t, "true && true")
	assertFalse(t, "true && false")
	assertFalse(t, "false && true")
	assertFalse(t, "false && false")
}

func TestList(t *testing.T) {
	assertInt(t, "[1, 2][0]", 1)
	assertInt(t, "[1, 2][1]", 2)

	assertInt(t, "(0 :: [1, 2])[0]", 0)
	assertInt(t, "(0 :: 1 :: 2 :: [])[1]", 1)

	assertInt(t, "[1, 2, 3][1:][0]", 2)
	assertInt(t, "[1, 2, 3][1:][1]", 3)
	assertInt(t, "[1, 2, 3][:2][0]", 1)
	assertInt(t, "[1, 2, 3][:2][1]", 2)
}

func TestMethodDef(t *testing.T) {
	assertInt(
		t,
		`
	fn (lst: List) head() {
		lst[0]
	}

	[1, 2].head()`,
		1)
}

func TestReturn(t *testing.T) {
	assertInt(t, `
	fn foo() { 
		return 3
		4
	}
	foo()
	`, 3)
}
