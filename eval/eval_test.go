package eval

import (
	"testing"

	"github.com/rymdhund/wosh/parser"
)

func runner(t *testing.T, prog string) *Runner {
	p := parser.NewParser(prog)
	exprs, err := p.Parse()
	if err != nil {
		t.Error(err)
	}
	return NewRunner(exprs)
}

func TestEvalAssign(t *testing.T) {
	r := runner(t, "a = 1")
	r.Run()
	o, _ := r.baseEnv.get("a")
	exp := Object{"int", 1}
	if o != exp {
		t.Errorf("Expected int(1), got %v", o)
	}
}

func TestEvalAssign2(t *testing.T) {
	r := runner(t, "a = 1 + 1\nb = a + 1\nc = a + b")
	r.Run()
	o, _ := r.baseEnv.get("a")
	exp := Object{"int", 2}
	if o != exp {
		t.Errorf("Expected int(2), got %v", o)
	}
	o, _ = r.baseEnv.get("b")
	exp = Object{"int", 3}
	if o != exp {
		t.Errorf("Expected int(3), got %v", o)
	}
	o, _ = r.baseEnv.get("c")
	exp = Object{"int", 5}
	if o != exp {
		t.Errorf("Expected int(5), got %v", o)
	}
}

func TestEvalMany(t *testing.T) {
	tests := []struct {
		string
		Object
	}{
		{"res = 1 + 2", IntVal(3)},
		{"a = 0\nres = 1 + 2", IntVal(3)},
		{"a = 0\n if a { res = 1 } else { res = 2 }", IntVal(2)},
		{`if 1 { res = 2 } else { res = 3 }`, IntVal(2)},
		{`res = if 1 { 2 } else { 3 }`, IntVal(2)},
		{`res = if 0 { 2 }`, UnitVal},
	}
	for _, test := range tests {
		prog, expected := test.string, test.Object
		r := runner(t, prog)
		r.Run()
		res, _ := r.baseEnv.get("res")
		if res != expected {
			t.Errorf("Got %s, expected %s", res, expected)
		}
	}
}
