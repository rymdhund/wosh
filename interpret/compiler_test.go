package interpret

import (
	"testing"

	"github.com/rymdhund/wosh/ast"
	"github.com/rymdhund/wosh/lexer"
	"github.com/rymdhund/wosh/parser"
)

func parseMain(prog string) (*ast.FuncDefExpr, error) {
	p := parser.NewParser(prog)
	exprs, _, err := p.Parse()
	if err != nil {
		return nil, err
	}
	return &ast.FuncDefExpr{
		Ident:      &ast.Ident{"main", lexer.Area{}},
		ClassParam: nil,
		Params:     []*ast.ParamExpr{},
		Body:       exprs,
		Area:       lexer.Area{},
	}, nil
}

func run(t *testing.T, prog string) Value {
	t.Helper()
	main, err := parseMain(prog)
	if err != nil {
		t.Fatalf("Parse error:%s", err)
	}
	function, err := Compile(main)
	if err != nil {
		t.Fatalf("Compile error:%s", err)
	}

	vm := NewVm()
	v, err := vm.Interpret(function)
	if err != nil {
		t.Fatalf("Run error:%s", err)
	}
	return v
}

func testEqual(v1, v2 Value) bool {
	return builtinEq(v1, v2).Val
}

func assertRes(t *testing.T, prog string, res Value) {
	t.Helper()
	main, err := parseMain(prog)
	if err != nil {
		t.Fatalf("Error parsing `%s`: %s", prog, err)
	}
	function, err := Compile(main)
	if err != nil {
		t.Fatalf("Error compiling `%s`: %s", prog, err)
	}

	vm := NewVm()
	v, err := vm.Interpret(function)
	if err != nil {
		t.Fatalf("Error running `%s`: %s", prog, err)
	}
	if !testEqual(res, v) {
		t.Errorf("Incorrect result on running `%s`, expected %s, got %s", prog, res, v)
	}
}

func assertRuntimeError(t *testing.T, prog string) {
	t.Helper()
	main, err := parseMain(prog)
	if err != nil {
		t.Fatalf("Error parsing `%s`: %s", prog, err)
	}
	function, err := Compile(main)
	if err != nil {
		t.Fatalf("Error compiling `%s`: %s", prog, err)
	}

	vm := NewVm()
	_, err = vm.Interpret(function)
	if err == nil {
		t.Errorf("Expected error when running `%s`", prog)
	}
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
	assertRes(t, "x = 1 + 1\nx", NewInt(2))
}

func TestLiterals(t *testing.T) {
	tests := []struct {
		string
		Value
	}{
		{"123", NewInt(123)},
		{"\"abc\"", NewString("abc")},
		{"'abc'", NewString("abc")},
		{"true", NewBool(true)},
		{"false", NewBool(false)},
		{"List", NewTypeValue(ListType)},
	}
	for _, test := range tests {
		prog, expected := test.string, test.Value
		assertRes(t, prog, expected)
	}
}
func TestIf(t *testing.T) {
	res := run(t, "if 1 == 1 { 2 } else { 3 }")
	if !testEqual(res, NewInt(2)) {
		t.Errorf("expected 2, got %s", res)
	}

	res = run(t, "if 1 == 2 { 3 } else { 4 }")
	if !testEqual(res, NewInt(4)) {
		t.Errorf("expected 4, got %s", res)
	}

	res = run(t, "x = 0 \n if 1 == 1 { x = 10 } \n x")
	if !testEqual(res, NewInt(10)) {
		t.Errorf("expected 10, got %s", res)
	}

	res = run(t, "x = 0 \n if 1 == 2 { x = 10 } \n x")
	if !testEqual(res, NewInt(0)) {
		t.Errorf("expected 0, got %s", res)
	}

	// Else If
	res = run(t, "if 1 == 1 { 1 } else if 1 == 1 { 2 } else { 3 }")
	if !testEqual(res, NewInt(1)) {
		t.Errorf("expected 1, got %s", res)
	}

	res = run(t, "if 1 == 2 { 1 } else if 1 == 1 { 2 } else { 3 }")
	if !testEqual(res, NewInt(2)) {
		t.Errorf("expected 2, got %s", res)
	}

	res = run(t, "if 1 == 2 { 1 } else if 2 == 1 { 2 } else { 3 }")
	if !testEqual(res, NewInt(3)) {
		t.Errorf("expected 3, got %s", res)
	}

	res = run(t, "if 1 == 2 { 1 } else if 2 == 1 { 2 } else if 3 == 3 { 3 } else { 4 }")
	if !testEqual(res, NewInt(3)) {
		t.Errorf("expected 3, got %s", res)
	}
}

func TestFor(t *testing.T) {
	res := run(t, "x = 1 \n for x == 1 { x = x + 1 } \n x")
	if !testEqual(res, NewInt(2)) {
		t.Errorf("expected 2, got %s", res)
	}

	res = run(t, "x = 1 \n for x < 10 { x = x + 1 } \n x")
	if !testEqual(res, NewInt(10)) {
		t.Errorf("expected 10, got %s", res)
	}

	res = run(t, "x = 1 \n for false { x = x + 1 } \n x")
	if !testEqual(res, NewInt(1)) {
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
	if !testEqual(v, Nil) {
		t.Errorf("Expected Nil, got %s", v)
	}
}

func TestCall(t *testing.T) {
	res := run(t, "fn x(y) { y + 1} \n x(4)")
	if !testEqual(res, NewInt(5)) {
		t.Errorf("expected 5, got %s", res)
	}

	res = run(t, "fn x(y) { y + 2} \n 1 + x(x(2))")
	if !testEqual(res, NewInt(7)) {
		t.Errorf("expected 7, got %s", res)
	}

	res = run(t, "fn x(y) { y + 2} \n 1 + x(x(2) + 1)")
	if !testEqual(res, NewInt(8)) {
		t.Errorf("expected 8, got %s", res)
	}

	res = run(t, "fn x(y) { z = y + 2 \n z + 1} \n x(1)")
	if !testEqual(res, NewInt(4)) {
		t.Errorf("expected 4, got %s", res)
	}
}

func TestArrow(t *testing.T) {
	run(t, `
	f = (a) => a + 1
	assert(f(1) == 2, "arrow1")

	g = (f) => f(1) + 1
	res = g((x) => x + 2)
	assert(res == 4, "arrow2")
	`)
}

func TestClosure(t *testing.T) {
	res := run(t, "fn f() { a = 1\n fn g() { a }\n g} \n b = f()\n b()")
	if !testEqual(res, NewInt(1)) {
		t.Errorf("expected 1, got %s", res)
	}

	res = run(t, "fn f() { a = 1\n fn g() { a = a + 1 }\n g()\n a} \n f()")
	if !testEqual(res, NewInt(2)) {
		t.Errorf("expected 2, got %s", res)
	}

	res = run(t, "fn f() { a = 1\n fn g() { a = 2 }\n g()\n a} \n f()")
	if !testEqual(res, NewInt(2)) {
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
	if !testEqual(res, NewInt(2)) {
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
	if !testEqual(res, NewInt(10)) {
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
	if !testEqual(res, NewInt(1)) {
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
	if !testEqual(res, NewInt(1)) {
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
	if !testEqual(res, NewInt(3)) {
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
	if !testEqual(res, NewInt(3)) {
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
	if !testEqual(res, NewInt(3)) {
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
	if !testEqual(res, NewInt(3)) {
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
	if !testEqual(res, NewInt(4)) {
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
	if !testEqual(res, NewInt(113)) {
		t.Errorf("expected 113, got %s", res)
	}
}

func assertFalse(t *testing.T, prog string) {
	t.Helper()
	res := run(t, prog)
	if !testEqual(res, NewBool(false)) {
		t.Errorf("expected false, got %s", res)
	}
}

func assertTrue(t *testing.T, prog string) {
	t.Helper()
	res := run(t, prog)
	if !testEqual(res, NewBool(true)) {
		t.Errorf("expected true, got %s", res)
	}
}

func assertInt(t *testing.T, prog string, value int) {
	t.Helper()
	res := run(t, prog)
	if !testEqual(res, NewInt(value)) {
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

	// test lazy
	assertFalse(t, "false && assert(false, '')")
	assertTrue(t, "true || assert(false, '')")
}

func TestList(t *testing.T) {
	assertInt(t, "[1, 2][0]", 1)
	assertInt(t, "[1, 2][1]", 2)
	assertInt(t, "[1, 2][-1]", 2)

	assertInt(t, "(0 :: [1, 2])[0]", 0)
	assertInt(t, "(0 :: 1 :: 2 :: [])[1]", 1)

	assertInt(t, "[1, 2, 3][1:][0]", 2)
	assertInt(t, "[1, 2, 3][1:][1]", 3)
	assertInt(t, "[1, 2, 3][:2][0]", 1)
	assertInt(t, "[1, 2, 3][:2][1]", 2)
	assertInt(t, "[1, 2, 3][:2][-1]", 2)

	assertInt(t, "[[1],[2]][0][0]", 1)
	assertInt(t, "[0, 1, 2][2]", 2)
	assertInt(t, "len([0, 1, 2])", 3)
	assertInt(t, "[0, 1, 2, 3][1:2][0]", 1)
	assertInt(t, "len([0, 1, 2, 3][1:2])", 1)
	assertInt(t, "[0, 1, 2, 3][1:2:1][0]", 1)
	assertInt(t, "len([0, 1, 2, 3][1:0])", 0)
	assertInt(t, "[0, 1, 2, 3][-2:-1][0]", 2)
	assertInt(t, "len([0, 1, 2, 3][-2:-1])", 1)
	assertInt(t, "len([0, 1][-100:1000])", 2)
	assertInt(t, "[0, 1, 2][1:][0]", 1)
	assertInt(t, "[0, 1, 2][:2][-1]", 1)
	assertInt(t, "[3, 4][:][0]", 3)
	assertInt(t, "[3, 4][:][1]", 4)
	assertInt(t, "len([0] + [1])", 2)
}

func TestMap(t *testing.T) {
	assertInt(t, "{'a': 1, 'b': 2}['a']", 1)
	assertTrue(t, "{'abc': true}['abc']")
	assertTrue(t, "{'abc': true}['abc']")

	run(t, `
	m = {"a": 1, "b": 2}

	assert(m["a"] == 1, "error 1")

	m["a"] = 2
	assert(m["a"] == 2, "error 2")
	`)
}

func TestMethodDef(t *testing.T) {
	run(t, `
	fn (lst: List) head() {
		lst[0]
	}

	assert([1, 2].head() == 1, "[1, 2].head() fail")

	x = List.head
	assert(x([1, 2]) == 1, "List.head fail 2")

	assert(List.head([1, 2]) == 1, "List.head fail 3")
	`)

	run(t, `
	fn (lst: List) eq(lst2) {
		len(lst) == len(lst2)
	}

	#fn (lst: List) add(lst2) {
	#	len(lst) + len(lst2)
	#}

	fn (lst: List) sub(lst2) {
		len(lst) - len(lst2)
	}

	fn (lst: List) mult(lst2) {
		println(lst)
		println(lst2)
		len(lst) * len(lst2)
	}
	x = [1, 2]
	y = [3, 4, 3]

	#assert(x + y == 5) cant override add
	assert(x - y == -1, "sub failed")
	assert(x * y == 6, "mult failed")
	assert([1, 2] == [3, 4], "eq failed")
	`)
}

func TestTypeDef(t *testing.T) {
	run(t, `
		type Foo(a: Int)
	
		fn (f: Foo) bar() {
			f.a + 1
		}
	
		x = Foo(3)
		assert(x.a == 3, "Typedef 1")
		assert(x.bar() == 4, "Typedef 2")
		`)

	run(t, `
		type Coord(x: Int, y: Int)

		fn (c: Coord) add(c2) {
		println("add")
		println(c)
		println(c2)
		Coord(c.x + c2.x, c.y + c2.y)
		}

		a = Coord(2, 2)
		b = Coord(3, 4)
		c = a + b
		assert(c.x == 5, "error")
	`)
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

func TestMatch(t *testing.T) {
	//run(t, `
	//x = match 3 {
	//	1 => "a"
	//	2 => "b"
	//	3 => "c"
	//}

	//assert(x == "c", "match error")
	//`)
}

func TestString(t *testing.T) {
	assertRes(t, "'abc' + 'def'", NewString("abcdef"))
	assertRes(t, "ord('a')", NewInt(97))
	assertRes(t, "'abc'[1:]", NewString("bc"))
	assertRes(t, "'åäö'[1:]", NewString("äö"))
	assertRes(t, "'åäö'[1]", NewString("ä"))
}

func TestDestructure(t *testing.T) {
	assertRes(t, "[x, y] = [1, 2]\nx", NewInt(1))
	assertRes(t, "[x, y] = [1, 2]\ny", NewInt(2))
	assertRes(t, "[[x], y] = [[1], 2]\nx", NewInt(1))
	assertRes(t, "[[a], b, [c, [d]]] = [[1], 2, [3, [4]]]\nd", NewInt(4))
	assertRuntimeError(t, "[x, y] = [1, 2, 3]")
	assertRuntimeError(t, "[x, y] = 'abc'")
}

func TestSmall(t *testing.T) {
	tests := []struct {
		string
		Value
	}{
		{"1 + 2", NewInt(3)},
		{"(1 + 2)", NewInt(3)},
		{"2 * 2", NewInt(4)},
		{"2 + 3 * 4 - 5", NewInt(9)},
		{"3 + 3 / 3", NewInt(4)},
		{"3 / 3 + 3", NewInt(4)},
		{"-2", NewInt(-2)},
		{"- -2", NewInt(2)},
		{"-(-2)", NewInt(2)},
		{"-2 - -2", NewInt(0)},
		{"4 - 2 - 2", NewInt(0)},
		{"4 / 2 / 2", NewInt(1)},
		{"4 % 2", NewInt(0)},
		{"5 % 2", NewInt(1)},
		{"'abc'", NewString("abc")},
		{"1 == 1", NewBool(true)},
		{"1 != 1", NewBool(false)},
		{"1 == 0", NewBool(false)},
		{"1 != 0", NewBool(true)},
		{"()", Nil},
		{"atoi('12')", NewInt(12)},
		{"'abc' + 'def'", NewString("abcdef")},
		//{"'one' + str(1)", NewString("one1")},
		{"'åäö'[1]", NewString("ä")},
		{"len('abc')", NewInt(3)},
		{"len('åäö')", NewInt(3)},
		{"len([1, 2, 3])", NewInt(3)},
		{"a = 0\n1 + 2", NewInt(3)},
		{"a = false\n if a { 1 } else { 2 }", NewInt(2)},
		{`if true { 2 } else { 3 }`, NewInt(2)},
		{`if 1 == 1 { 2 } else { 3 }`, NewInt(2)},
		{`if false { 2 }`, Nil},
		{"println('abc')", Nil},
		{"a = 1 # test\n# comment\nb=a #comment\nb#comment", NewInt(1)},
		//{"res <- echo('abc')", NewString("abc\n")},
		//{"res <-2 echo_err('abc')", NewString("abc\n")},
		//{"res <- `echo abc`", NewString("abc\n")},
		//{"res <-2 `../utils/echo_err.sh eee`", NewString("eee\n")},
	}
	for _, test := range tests {
		prog, expected := test.string, test.Value
		assertRes(t, prog, expected)
	}
}
