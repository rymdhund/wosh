package eval

import (
	"testing"

	. "github.com/rymdhund/wosh/obj"
	"github.com/rymdhund/wosh/parser"
)

func runner(t *testing.T, prog string) *Runner {
	p := parser.NewParser(prog)
	exprs, err := p.Parse()
	if err != nil {
		t.Fatalf("Parsing error: %s", err)
	}
	return NewRunner(exprs)
}

func TestEvalAssign(t *testing.T) {
	r := runner(t, "a = 1")
	r.Run()
	o, _ := r.baseEnv.get("a")
	if !Equal(o, IntVal(1)) {
		t.Errorf("Expected int(1), got %v", o)
	}
}

func TestEvalAssign2(t *testing.T) {
	r := runner(t, "a = 1 + 1\nb = a + 1\nc = a + b")
	r.Run()
	o, _ := r.baseEnv.get("a")
	if !Equal(o, IntVal(2)) {
		t.Errorf("Expected int(2), got %v", o)
	}
	o, _ = r.baseEnv.get("b")
	if !Equal(o, IntVal(3)) {
		t.Errorf("Expected int(3), got %v", o)
	}
	o, _ = r.baseEnv.get("c")
	if !Equal(o, IntVal(5)) {
		t.Errorf("Expected int(5), got %v", o)
	}
}

func TestEvalList(t *testing.T) {
	r := runner(t, "a = []\nc = [1, 2] \nb = c[0]")
	r.Run()
	o, _ := r.baseEnv.get("a")
	if !Equal(o, ListNil()) {
		t.Errorf("Expected list(), got %v", o)
	}
	o, _ = r.baseEnv.get("c")
	if !Equal(o, ListVal(IntVal(1), ListVal(IntVal(2), ListNil()))) {
		t.Errorf("Expected [1, 2], got %v", o)
	}
	o, _ = r.baseEnv.get("b")
	if !Equal(o, IntVal(1)) {
		t.Errorf("Expected 1, got %v", o)
	}
}

func TestTrailingCommaInList(t *testing.T) {
	r := runner(t, "a = [\n1,\n2,\n]")
	r.Run()
	o, _ := r.baseEnv.get("a")
	if !Equal(o, ListVal(IntVal(1), ListVal(IntVal(2), ListNil()))) {
		t.Errorf("Expected [1, 2], got %v", o)
	}
}

func TestEvalMany(t *testing.T) {
	tests := []struct {
		string
		Object
	}{
		{"res = 1 + 2", IntVal(3)},
		{"res = (1 + 2)", IntVal(3)},
		{"res = 2 * 2", IntVal(4)},
		{"res = 2 + 3 * 4 - 5", IntVal(9)},
		{"res = 3 + 3 / 3", IntVal(4)},
		{"res = 3 / 3 + 3", IntVal(4)},
		{"res = -2", IntVal(-2)},
		{"res = - -2", IntVal(2)},
		{"res = -(-2)", IntVal(2)},
		{"res = -2 - -2", IntVal(0)},
		{"res = 4 - 2 - 2", IntVal(0)},
		{"res = 4 / 2 / 2", IntVal(1)},
		{"res = 'abc'", StrVal("abc")},
		{"res = 1 == 1", BoolVal(true)},
		{"res = 1 != 1", BoolVal(false)},
		{"res = 1 == 0", BoolVal(false)},
		{"res = 1 != 0", BoolVal(true)},
		{"res = ()", UnitVal},
		{"res = int('12')", IntVal(12)},
		{"res = 'abc' + 'def'", StrVal("abcdef")},
		{"res = 'one' + str(1)", StrVal("one1")},
		{"res = 'åäö'[1]", StrVal("ä")},
		{"res = len('abc')", IntVal(3)},
		{"res = len('åäö')", IntVal(3)},
		{"res = len([1, 2, 3])", IntVal(3)},
		{"a = 0\nres = 1 + 2", IntVal(3)},
		{"a = false\n if a { res = 1 } else { res = 2 }", IntVal(2)},
		{`if true { res = 2 } else { res = 3 }`, IntVal(2)},
		{`res = if 1 == 1 { 2 } else { 3 }`, IntVal(2)},
		{`res = if false { 2 }`, UnitVal},
		{"res <- echo('abc')", StrVal("abc\n")},
		{"res <-2 echo_err('abc')", StrVal("abc\n")},
		{"res = echo('abc')", UnitVal},
		{"a = 1 # test\n# comment\nb=a #comment\nres=b#comment", IntVal(1)},
		{"res <- `echo abc`", StrVal("abc\n")},
		{"res <-2 `../utils/echo_err.sh eee`", StrVal("eee\n")},
	}
	for _, test := range tests {
		prog, expected := test.string, test.Object
		r := runner(t, prog)
		err := r.Run()
		if err != nil {
			t.Fatal(err)
		}
		res, _ := r.baseEnv.get("res")
		if !Equal(res, expected) {
			t.Errorf("Got %s, expected %s", res, expected)
		}
	}
}

func TestBool(t *testing.T) {
	tests := []struct {
		string
		Object
	}{
		{"res = true", BoolVal(true)},
		{"res = false", BoolVal(false)},
		{"res = 1 > 0", BoolVal(true)},
		{"res = 1 < 0", BoolVal(false)},
		{"res = true && true", BoolVal(true)},
		{"res = true && false", BoolVal(false)},
		{"res = true || false", BoolVal(true)},
		{"res = false || false", BoolVal(false)},
	}
	for _, test := range tests {
		prog, expected := test.string, test.Object
		r := runner(t, prog)
		err := r.Run()
		if err != nil {
			t.Fatal(err)
		}
		res, _ := r.baseEnv.get("res")
		if !Equal(res, expected) {
			t.Errorf("Got %s, expected %s", res, expected)
		}
	}
}

func TestList(t *testing.T) {
	tests := []struct {
		string
		Object
	}{
		{"res = [[1],[2]][0][0]", IntVal(1)},
		{"res = [0, 1, 2][2]", IntVal(2)},
		{"res = len([0, 1, 2])", IntVal(3)},
		{"res = [0, 1, 2, 3][1:2]", ListVal(IntVal(1), ListNil())},
		{"res = [0, 1, 2, 3][1:2:1]", ListVal(IntVal(1), ListNil())},
		{"res = [0, 1, 2, 3][1:0]", ListNil()},
		{"res = [0, 1, 2, 3][-2:-1]", ListVal(IntVal(2), ListNil())},
		{"res = [0, 1][-100:1000]", ListVal(IntVal(0), ListVal(IntVal(1), ListNil()))},
		{"res = [0, 1, 2][1:]", ListVal(IntVal(1), ListVal(IntVal(2), ListNil()))},
		{"res = [0, 1, 2][:2]", ListVal(IntVal(0), ListVal(IntVal(1), ListNil()))},
		{"res = [0, 1][:]", ListVal(IntVal(0), ListVal(IntVal(1), ListNil()))},
		{"res = 'abc'[1:]", StrVal("bc")},
		{"res = [0] + [1]", ListVal(IntVal(0), ListVal(IntVal(1), ListNil()))},
	}
	for _, test := range tests {
		prog, expected := test.string, test.Object
		r := runner(t, prog)
		err := r.Run()
		if err != nil {
			t.Fatal(err)
		}
		res, _ := r.baseEnv.get("res")
		if !Equal(res, expected) {
			t.Errorf("Got %s, expected %s", res, expected)
		}
	}
}

func TestMap(t *testing.T) {
	tests := []struct {
		string
		Object
	}{
		{"res = {'a': 1}['a']", IntVal(1)},
		{"x = {}\n map_set(x, 'a', 1)\n res = x['a']", IntVal(1)},
	}
	for _, test := range tests {
		prog, expected := test.string, test.Object
		r := runner(t, prog)
		err := r.Run()
		if err != nil {
			t.Fatal(err)
		}
		res, _ := r.baseEnv.get("res")
		if !Equal(res, expected) {
			t.Errorf("Got %s, expected %s", res, expected)
		}
	}
}

func TestMethod(t *testing.T) {
	tests := []struct {
		string
		Object
	}{
		{"fn (s: Str) abc() { 1 }\n res = ''.abc()", IntVal(1)},
	}
	for _, test := range tests {
		prog, expected := test.string, test.Object
		r := runner(t, prog)
		err := r.Run()
		if err != nil {
			t.Fatal(err)
		}
		res, _ := r.baseEnv.get("res")
		if !Equal(res, expected) {
			t.Errorf("Got %s, expected %s", res, expected)
		}
	}
}

func TestEvalError(t *testing.T) {
	tests := []struct {
		prog string
		exp  string
	}{
		{"res <-? raise('test')", "Exception"},
		{"res <-? `diff`", "Exception"},
		{"res <-? if true { raise(2) }", "Exception"},
	}
	for _, test := range tests {
		prog, expected := test.prog, test.exp
		r := runner(t, prog)
		r.Run()
		res, _ := r.baseEnv.get("res")
		if res.Class().Name != expected {
			t.Errorf("Got %s, expected %s", res.Class().Name, expected)
		}
	}
}

func TestEvalFunc(t *testing.T) {
	r := runner(t, `
	fn foo(x) {
		x + 1
	}
	res = foo(2)
	`)
	r.Run()
	o, _ := r.baseEnv.get("res")
	if !Equal(o, IntVal(3)) {
		t.Errorf("Expected int(3), got %v", o)
	}
}

func TestFuncNoParam(t *testing.T) {
	r := runner(t, `
	x = 3
	fn foo() {
		x + 1
	}
	res = foo()
	`)
	r.Run()
	o, _ := r.baseEnv.get("res")
	if !Equal(o, IntVal(4)) {
		t.Errorf("Expected int(4), got %v", o)
	}
}

func TestEvalFor(t *testing.T) {
	r := runner(t, `
	x = 3
	for x != 0 {
		x = x - 1
	}
	`)
	r.Run()
	o, _ := r.baseEnv.get("x")
	if !Equal(o, IntVal(0)) {
		t.Errorf("Expected int(0), got %v", o)
	}
}
