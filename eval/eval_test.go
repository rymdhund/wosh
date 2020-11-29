package eval

import (
	"testing"

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

func TestEvalMany(t *testing.T) {
	tests := []struct {
		string
		Object
	}{
		{"res = 1 + 2", IntVal(3)},
		{"res = (1 + 2)", IntVal(3)},
		{"res = 'abc'", StrVal("abc")},
		{"a = 0\nres = 1 + 2", IntVal(3)},
		{"a = 0\n if a { res = 1 } else { res = 2 }", IntVal(2)},
		{`if 1 { res = 2 } else { res = 3 }`, IntVal(2)},
		{`res = if 1 { 2 } else { 3 }`, IntVal(2)},
		{`res = if 0 { 2 }`, UnitVal},
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
		r.Run()
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
		{"res <-? raise('test')", "exception"},
		{"res <-? `diff`", "exit"},
		{"res <-? if 1 { raise(2) }", "exception"},
	}
	for _, test := range tests {
		prog, expected := test.prog, test.exp
		r := runner(t, prog)
		r.Run()
		res, _ := r.baseEnv.get("res")
		if res.Type() != expected {
			t.Errorf("Got %s, expected %s", res.Type(), expected)
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
